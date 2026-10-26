package agent

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

// DetectedInstance is one database engine found listening on this machine.
// Only protocol-level facts are reported — no credentials are used or needed
// to identify a server, and nothing is stored.
type DetectedInstance struct {
	Type    string `json:"type"`    // POSTGRES | MONGODB | MSSQL (ConnectionType values)
	Product string `json:"product"` // display label
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Source  string `json:"source"` // how it was identified, for the UI
}

// DetectResponse is the payload of GET /detect.
type DetectResponse struct {
	Instances []DetectedInstance `json:"instances"`
	Scanned   []int              `json:"scanned_ports"`
	Error     string             `json:"error,omitempty"`
}

// Ports probed on the loopback interface. These are the defaults for the
// engines VaultGuard can back up; servers on unusual or dynamic ports (for
// example a named SQL Server instance) are not guessed at — the connection
// test finds those.
var detectPorts = []int{5432, 5433, 1433, 27017, 27018}

const detectHost = "127.0.0.1"

// probeTimeout bounds each dial and each protocol exchange so a filtered or
// non-responsive port never stalls the whole scan.
const probeTimeout = 400 * time.Millisecond

func handleDetect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Same loopback-helper policy as /browse: any local dashboard origin.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "use GET"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	out := DetectResponse{Instances: []DetectedInstance{}, Scanned: detectPorts}
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, port := range detectPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			if inst, ok := identifyPort(ctx, p); ok {
				mu.Lock()
				out.Instances = append(out.Instances, inst)
				mu.Unlock()
			}
		}(port)
	}
	wg.Wait()

	sort.Slice(out.Instances, func(i, j int) bool { return out.Instances[i].Port < out.Instances[j].Port })
	if ctx.Err() != nil {
		out.Error = "scan timed out before finishing every port"
	}
	json.NewEncoder(w).Encode(out)
}

// identifyPort dials the port once per candidate protocol and returns the
// engine that actually answers. Each probe speaks its own wire protocol, so
// an unrelated service on the same port is never misreported.
func identifyPort(ctx context.Context, port int) (DetectedInstance, bool) {
	addr := net.JoinHostPort(detectHost, strconv.Itoa(port))
	product := func(name, kind string) DetectedInstance {
		return DetectedInstance{Type: kind, Product: name, Host: detectHost, Port: port, Source: "protocol handshake"}
	}
	switch {
	case probePostgres(ctx, addr):
		return product("PostgreSQL", "POSTGRES"), true
	case probeMSSQL(ctx, addr):
		return product("SQL Server", "MSSQL"), true
	case probeMongo(ctx, addr):
		return product("MongoDB", "MONGODB"), true
	}
	return DetectedInstance{}, false
}

func dial(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: probeTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(probeTimeout))
	return conn, nil
}

// probePostgres sends a frontend SSLRequest and expects the one-byte reply
// every PostgreSQL backend sends ('S' or 'N' — anything else is not PG).
func probePostgres(ctx context.Context, addr string) bool {
	conn, err := dial(ctx, addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	req := make([]byte, 8)
	binary.BigEndian.PutUint32(req[0:4], 8)        // message length
	binary.BigEndian.PutUint32(req[4:8], 80877103) // SSLRequest code
	if _, err := conn.Write(req); err != nil {
		return false
	}
	reply := make([]byte, 1)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return false
	}
	return reply[0] == 'S' || reply[0] == 'N'
}

// probeMSSQL sends a TDS PRELOGIN packet; a SQL Server answers with a
// PRELOGINRESPONSE (packet type 0x04).
func probeMSSQL(ctx context.Context, addr string) bool {
	conn, err := dial(ctx, addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Header (8 bytes) + one ENCRYPTION option (token/offset/length) +
	// 0xFF terminator + one data byte. Offsets are from the packet start.
	const (
		headerLen  = 8
		tableLen   = 6 // 5-byte option entry + terminator
		dataOffset = headerLen + tableLen
	)
	pkt := make([]byte, dataOffset+1)
	pkt[0] = 0x12 // PRELOGIN
	pkt[1] = 0x01 // EOM
	binary.BigEndian.PutUint16(pkt[2:4], uint16(len(pkt)))
	pkt[6] = 0x01 // packet id
	pkt[8] = 0x01 // ENCRYPTION option token
	binary.BigEndian.PutUint16(pkt[9:11], dataOffset)
	binary.BigEndian.PutUint16(pkt[11:13], 1)
	pkt[13] = 0xFF         // end of option table
	pkt[dataOffset] = 0x00 // ENCRYPT_OFF

	if _, err := conn.Write(pkt); err != nil {
		return false
	}
	reply := make([]byte, 8)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return false
	}
	return reply[0] == 0x04
}

// probeMongo sends an OP_MSG "isMaster" (the wire handshake every mongod
// answers without authentication). A reply with a Mongo opcode confirms it.
func probeMongo(ctx context.Context, addr string) bool {
	conn, err := dial(ctx, addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	doc := bsonDoc(
		bsonBool("isMaster", true),
		bsonString("$db", "admin"),
	)
	const opMsg = 2013
	const headerLen = 16
	msg := make([]byte, 0, headerLen+4+1+len(doc))
	put32 := func(v uint32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], v)
		msg = append(msg, b[:]...)
	}
	put32(uint32(headerLen + 4 + 1 + len(doc))) // total length
	put32(1)                                    // request id
	put32(0)                                    // response to
	put32(opMsg)                                // opcode
	put32(0)                                    // flag bits
	msg = append(msg, 0x00)                     // section kind 0 (single document)
	msg = append(msg, doc...)

	if _, err := conn.Write(msg); err != nil {
		return false
	}
	head := make([]byte, headerLen)
	if _, err := io.ReadFull(conn, head); err != nil {
		return false
	}
	op := binary.LittleEndian.Uint32(head[12:16])
	return op == opMsg || op == 2004 // OP_MSG or legacy OP_REPLY
}

// bsonBool appends a single boolean element.
func bsonBool(name string, v bool) []byte {
	b := append([]byte{0x08}, name...)
	b = append(b, 0x00)
	if v {
		return append(b, 0x01)
	}
	return append(b, 0x00)
}

// bsonString appends a single UTF-8 string element.
func bsonString(name, v string) []byte {
	b := append([]byte{0x02}, name...)
	b = append(b, 0x00)
	var l [4]byte
	binary.LittleEndian.PutUint32(l[:], uint32(len(v)+1)) // includes the NUL
	b = append(b, l[:]...)
	b = append(b, v...)
	return append(b, 0x00)
}

// bsonDoc frames elements with a total-length prefix and terminator.
func bsonDoc(elems ...[]byte) []byte {
	body := []byte{}
	for _, e := range elems {
		body = append(body, e...)
	}
	body = append(body, 0x00) // document terminator
	var l [4]byte
	binary.LittleEndian.PutUint32(l[:], uint32(len(body)+4))
	return append(l[:], body...)
}
