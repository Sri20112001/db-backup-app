// Package realtime is the WebSocket event bus for live dashboard updates.
//
// Two client kinds share one endpoint (GET /vaultguard/api/ws):
//   - dashboards authenticate with a user JWT (?token=) and then send
//     {type:"subscribe", org_id} for organizations they belong to;
//   - agents authenticate with X-Agent-ID + Bearer agent token and receive
//     targeted commands (e.g. cancel) plus relay log lines to dashboards.
//
// The package-level DefaultHub is set once by main and used by handlers,
// the health monitor, and the gRPC server. All Publish paths are nil-safe
// so unit tests and alternative wirings keep working without a hub.
package realtime

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// DefaultHub is wired by main; everything else publishes through it.
var DefaultHub *Hub

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 64 * 1024
)

// Event types flowing to dashboards.
const (
	TypeRun      = "run"      // backup run created/progressed/finished/cancelled
	TypePresence = "presence" // agent ONLINE/OFFLINE flip
	TypeAlert    = "alert"    // new alert row
	TypeJobs     = "jobs"     // backup job created/updated/deleted/toggled
	TypeRestores = "restores" // restore created/finished
	TypeAgents   = "agents"   // agent registered/removed
	TypeLog      = "log"      // agent lifecycle/error line for a run
	TypeCancel   = "cancel"   // server -> agent: abort this run now
)

// Event is the wire envelope in both directions.
type Event struct {
	Type    string      `json:"type"`
	OrgID   string      `json:"org_id,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type peer struct {
	conn    *websocket.Conn
	send    chan []byte
	userID  uuid.UUID // set for dashboards
	agentID uuid.UUID // set for agents
	orgID   uuid.UUID // agent's org
	orgs    map[string]bool
	isAgent bool
}

type Hub struct {
	db          *gorm.DB
	jwtSecret   string
	allowOrigin string

	mu         sync.RWMutex
	orgPeers   map[string]map[*peer]struct{}
	agentPeers map[string]*peer
}

func NewHub(db *gorm.DB, jwtSecret, allowOrigin string) *Hub {
	return &Hub{
		db:          db,
		jwtSecret:   jwtSecret,
		allowOrigin: allowOrigin,
		orgPeers:    make(map[string]map[*peer]struct{}),
		agentPeers:  make(map[string]*peer),
	}
}

// upgraderFor returns a WebSocket upgrader that accepts the configured
// dashboard origin(s), preventing Cross-Site WebSocket Hijacking.
// CORS_ORIGIN may be comma-separated; loopback origins are always allowed
// so localhost vs 127.0.0.1 vs LAN-host dev servers all work.
func (h *Hub) upgraderFor() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Agents connect server-to-server with no Origin header.
			if origin == "" {
				return true
			}
			return wsOriginAllowed(origin, h.allowOrigin)
		},
	}
}

func wsOriginAllowed(origin, allowlist string) bool {
	if strings.TrimSpace(allowlist) == "*" {
		return true
	}
	norm := strings.TrimSuffix(strings.TrimSpace(origin), "/")
	for _, p := range strings.Split(allowlist, ",") {
		if v := strings.TrimSpace(strings.TrimSuffix(p, "/")); v != "" && (norm == v || v == "*") {
			return true
		}
	}
	lower := strings.ToLower(norm)
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		if strings.HasPrefix(lower, "http://"+host) || strings.HasPrefix(lower, "https://"+host) {
			return true
		}
	}
	return false
}

// Publish sends ev to every dashboard subscribed to orgID. Nil-hub safe.
func Publish(h *Hub, orgID string, ev Event) {
	if h == nil || orgID == "" {
		return
	}
	ev.OrgID = orgID
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for p := range h.orgPeers[orgID] {
		select {
		case p.send <- data:
		default:
			go p.closeSlow()
		}
	}
}

// SendToAgent delivers ev to the connected agent, if any. Nil-hub safe.
func SendToAgent(h *Hub, agentID string, ev Event) {
	if h == nil || agentID == "" {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	p, ok := h.agentPeers[agentID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case p.send <- data:
	default:
	}
}

// ServeWS upgrades the connection and routes by credential kind.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	if agentID, ok := h.authAgent(r); ok {
		h.serveAgent(w, r, agentID)
		return
	}
	// Dashboards can't set headers in `new WebSocket()`, so the JWT rides
	// the query string instead (?token=). Never logged.
	userID, ok := h.authUser(r.URL.Query().Get("token"))
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	h.serveDashboard(w, r, userID)
}

func (h *Hub) authUser(token string) (uuid.UUID, bool) {
	if token == "" || h.jwtSecret == "" {
		return uuid.Nil, false
	}
	claims := &struct {
		UserID string `json:"user_id"`
		jwt.RegisteredClaims
	}{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func (h *Hub) authAgent(r *http.Request) (uuid.UUID, bool) {
	raw := r.Header.Get("Authorization")
	if len(raw) <= 7 || raw[:7] != "Bearer " {
		return uuid.Nil, false
	}
	agentID, err := uuid.Parse(r.Header.Get("X-Agent-ID"))
	if err != nil {
		return uuid.Nil, false
	}
	var agent models.Agent
	if err := h.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		return uuid.Nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(agent.TokenHash), []byte(raw[7:])) != nil {
		return uuid.Nil, false
	}
	return agent.ID, true
}

func (h *Hub) upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, bool) {
	upgrader := h.upgraderFor()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug().Err(err).Msg("realtime: upgrade failed")
		return nil, false
	}
	return conn, true
}

func (h *Hub) serveDashboard(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	conn, ok := h.upgrade(w, r)
	if !ok {
		return
	}
	p := &peer{conn: conn, send: make(chan []byte, 64), userID: userID, orgs: map[string]bool{}}
	go p.writePump()
	p.readLoop(h, false)
	h.removePeer(p)
}

func (h *Hub) serveAgent(w http.ResponseWriter, r *http.Request, agentID uuid.UUID) {
	var agent models.Agent
	if err := h.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, ok := h.upgrade(w, r)
	if !ok {
		return
	}
	p := &peer{conn: conn, send: make(chan []byte, 64), agentID: agentID, orgID: agent.OrganizationID, isAgent: true}
	h.mu.Lock()
	if old, exists := h.agentPeers[agentID.String()]; exists {
		old.closeSlow()
	}
	h.agentPeers[agentID.String()] = p
	h.mu.Unlock()
	log.Info().Str("agent_id", agentID.String()).Msg("realtime: agent socket connected")
	go p.writePump()
	p.readLoop(h, true)
	h.mu.Lock()
	if h.agentPeers[agentID.String()] == p {
		delete(h.agentPeers, agentID.String())
	}
	h.mu.Unlock()
	h.removePeer(p)
	log.Info().Str("agent_id", agentID.String()).Msg("realtime: agent socket closed")
}

func (h *Hub) removePeer(p *peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for org := range p.orgs {
		if set, ok := h.orgPeers[org]; ok {
			delete(set, p)
			if len(set) == 0 {
				delete(h.orgPeers, org)
			}
		}
	}
}

// handleMessage routes one inbound frame.
func (h *Hub) handleMessage(p *peer, raw []byte) {
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		return
	}
	switch {
	case !p.isAgent && ev.Type == "subscribe":
		h.subscribe(p, ev.OrgID)
	case p.isAgent && ev.Type == TypeLog:
		h.relayAgentLog(p, ev.Payload)
	}
}

func (h *Hub) subscribe(p *peer, orgID string) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return
	}
	var n int64
	h.db.Model(&models.OrganizationMember{}).
		Where("organization_id = ? AND user_id = ?", id, p.userID).
		Count(&n)
	if n == 0 {
		return
	}
	h.mu.Lock()
	set, ok := h.orgPeers[orgID]
	if !ok {
		set = make(map[*peer]struct{})
		h.orgPeers[orgID] = set
	}
	set[p] = struct{}{}
	h.mu.Unlock()
	p.orgs[orgID] = true
}

// relayAgentLog forwards an agent's log line to its org dashboards after
// verifying the run belongs to that agent.
func (h *Hub) relayAgentLog(p *peer, payload interface{}) {
	m, ok := payload.(map[string]interface{})
	if !ok {
		return
	}
	runID, _ := m["run_id"].(string)
	line, _ := m["line"].(string)
	if runID == "" || line == "" {
		return
	}
	id, err := uuid.Parse(runID)
	if err != nil {
		return
	}
	var n int64
	h.db.Model(&models.BackupRun{}).Where("id = ? AND agent_id = ?", id, p.agentID).Count(&n)
	if n == 0 {
		return
	}
	Publish(h, p.orgID.String(), Event{Type: TypeLog, Payload: map[string]interface{}{
		"run_id":   runID,
		"agent_id": p.agentID.String(),
		"line":     line,
		"at":       time.Now().UTC().Format(time.RFC3339),
	}})
}

func (p *peer) readLoop(h *Hub, _ bool) {
	defer p.conn.Close()
	p.conn.SetReadLimit(maxMessageSize)
	p.conn.SetReadDeadline(time.Now().Add(pongWait))
	p.conn.SetPongHandler(func(string) error {
		p.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, raw, err := p.conn.ReadMessage()
		if err != nil {
			return
		}
		h.handleMessage(p, raw)
	}
}

func (p *peer) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.conn.Close()
	}()
	for {
		select {
		case data, ok := <-p.send:
			p.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				p.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := p.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			p.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := p.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (p *peer) closeSlow() {
	p.conn.Close()
}
