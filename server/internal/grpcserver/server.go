package grpcserver

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/backup-saas/server/internal/handlers"
	"github.com/backup-saas/server/internal/models"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type agentConn struct {
	agentID string
	send    chan *pb.ServerCommand
	cancel  context.CancelFunc
}

type Server struct {
	pb.UnimplementedAgentServiceServer
	db            *gorm.DB
	mu            sync.RWMutex
	streams        map[string]*agentConn
	encryptionKey  []byte
}

func NewServer(db *gorm.DB, encryptionKey []byte) *Server {
	return &Server{
		db:            db,
		streams:       make(map[string]*agentConn),
		encryptionKey: encryptionKey,
	}
}

// IsAgentConnected reports whether an agent has an active stream.
func (s *Server) IsAgentConnected(agentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.streams[agentID]
	return ok
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	pb.RegisterAgentServiceServer(srv, s)
	log.Info().Str("addr", addr).Msg("gRPC server listening")
	return srv.Serve(lis)
}

func (s *Server) Connect(stream pb.AgentService_ConnectServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}

	agent, err := s.authenticateAgent(first.AgentId, first.AgentToken)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(stream.Context())
	conn := &agentConn{
		agentID: agent.ID.String(),
		send:    make(chan *pb.ServerCommand, 16),
		cancel:  cancel,
	}
	s.mu.Lock()
	s.streams[agent.ID.String()] = conn
	s.mu.Unlock()

	now := time.Now()
	s.db.Model(agent).Updates(map[string]interface{}{
		"status":       models.AgentOnline,
		"last_seen_at": &now,
	})
	log.Info().Str("agent_id", agent.ID.String()).Msg("agent connected")

	defer func() {
		s.mu.Lock()
		delete(s.streams, agent.ID.String())
		s.mu.Unlock()
		s.db.Model(agent).Update("status", models.AgentOffline)
		cancel()
		log.Info().Str("agent_id", agent.ID.String()).Msg("agent disconnected")
	}()

	s.handleEvent(agent, first)

	recvErr := make(chan error, 1)
	go func() {
		for {
			evt, err := stream.Recv()
			if err != nil {
				recvErr <- err
				return
			}
			s.handleEvent(agent, evt)
		}
	}()

	for {
		select {
		case cmd := <-conn.send:
			if err := stream.Send(cmd); err != nil {
				return err
			}
		case err := <-recvErr:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Server) handleEvent(agent *models.Agent, evt *pb.AgentEvent) {
	now := time.Now()
	s.db.Model(agent).Update("last_seen_at", &now)

	switch p := evt.Payload.(type) {
	case *pb.AgentEvent_Connected:
		s.db.Model(agent).Updates(map[string]interface{}{
			"name":    p.Connected.Hostname,
			"version": p.Connected.Version,
		})
	case *pb.AgentEvent_Health:
		log.Debug().Str("agent_id", agent.ID.String()).Bool("healthy", p.Health.Healthy).Msg("agent health")

	case *pb.AgentEvent_BackupStarted:
		s.db.Model(&models.BackupRun{}).Where("id = ?", p.BackupStarted.RunId).
			Updates(map[string]interface{}{"status": models.RunRunning, "started_at": &now})

	case *pb.AgentEvent_BackupProgress:
		s.db.Model(&models.BackupRun{}).Where("id = ?", p.BackupProgress.RunId).
			Updates(map[string]interface{}{
				"status":           models.RunUploading,
				"bytes_read":       p.BackupProgress.BytesRead,
				"bytes_compressed": p.BackupProgress.BytesCompressed,
				"bytes_uploaded":   p.BackupProgress.BytesUploaded,
			})

	case *pb.AgentEvent_BackupCompleted:
		s.db.Model(&models.BackupRun{}).Where("id = ?", p.BackupCompleted.RunId).
			Updates(map[string]interface{}{
				"status":           models.RunCompleted,
				"completed_at":     &now,
				"bytes_read":       p.BackupCompleted.BytesRead,
				"bytes_compressed": p.BackupCompleted.BytesCompressed,
				"bytes_uploaded":   p.BackupCompleted.BytesUploaded,
				"checksum":         p.BackupCompleted.Checksum,
				"storage_path":     p.BackupCompleted.StoragePath,
			})

	case *pb.AgentEvent_BackupFailed:
		var run models.BackupRun
		s.db.Where("id = ?", p.BackupFailed.RunId).First(&run)
		s.db.Model(&models.BackupRun{}).Where("id = ?", p.BackupFailed.RunId).
			Updates(map[string]interface{}{
				"status":        models.RunFailed,
				"completed_at":  &now,
				"error_message": p.BackupFailed.Error,
			})
		agentID := agent.ID
		alert := models.Alert{
			OrganizationID: agent.OrganizationID,
			Type:           models.AlertBackupFailed,
			Title:          "Backup Failed",
			Message:        p.BackupFailed.Error,
			AgentID:        &agentID,
		}
		if run.ID != uuid.Nil {
			alert.BackupJobID = &run.BackupJobID
		}
		s.db.Create(&alert)

	case *pb.AgentEvent_RestoreStarted:
		s.db.Model(&models.RestoreJob{}).Where("id = ?", p.RestoreStarted.RestoreId).
			Updates(map[string]interface{}{"status": models.RestoreRunning, "started_at": &now})

	case *pb.AgentEvent_RestoreCompleted:
		s.db.Model(&models.RestoreJob{}).Where("id = ?", p.RestoreCompleted.RestoreId).
			Updates(map[string]interface{}{"status": models.RestoreCompleted, "completed_at": &now})

	case *pb.AgentEvent_RestoreFailed:
		s.db.Model(&models.RestoreJob{}).Where("id = ?", p.RestoreFailed.RestoreId).
			Updates(map[string]interface{}{
				"status":        models.RestoreFailed,
				"completed_at":  &now,
				"error_message": p.RestoreFailed.Error,
			})
	}
}

func (s *Server) ReportProgress(_ context.Context, req *pb.ProgressRequest) (*pb.ProgressResponse, error) {
	if _, err := s.authenticateAgent(req.AgentId, req.AgentToken); err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"bytes_read":       req.BytesRead,
		"bytes_compressed": req.BytesCompressed,
		"bytes_uploaded":   req.BytesUploaded,
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Status == string(models.RunCompleted) || req.Status == string(models.RunFailed) {
		updates["completed_at"] = &now
	}
	if req.Checksum != "" {
		updates["checksum"] = req.Checksum
	}
	if req.StoragePath != "" {
		updates["storage_path"] = req.StoragePath
	}
	if req.Error != "" {
		updates["error_message"] = req.Error
	}
	s.db.Model(&models.BackupRun{}).Where("id = ?", req.RunId).Updates(updates)
	return &pb.ProgressResponse{Accepted: true}, nil
}

func (s *Server) GetJobConfig(_ context.Context, req *pb.JobConfigRequest) (*pb.JobConfigResponse, error) {
	agent, err := s.authenticateAgent(req.AgentId, req.AgentToken)
	if err != nil {
		return nil, err
	}

	var job models.BackupJob
	if err := s.db.Preload("Schedule").Preload("StorageTarget").
		Where("id = ? AND agent_id = ?", req.JobId, agent.ID).First(&job).Error; err != nil {
		return nil, err
	}

	resp := &pb.JobConfigResponse{
		JobId:           job.ID.String(),
		SourceType:      string(job.SourceType),
		SourcePath:      job.SourcePath,
		SourceDatabase:  job.SourceDatabase,
		IncludePatterns: job.IncludePatterns,
		ExcludePatterns: job.ExcludePatterns,
		Mode:            string(job.Mode),
		Encrypted:       job.Encrypted,
		StorageType:     string(job.StorageTarget.Type),
		StorageBucket:   job.StorageTarget.Bucket,
		StorageRegion:   job.StorageTarget.Region,
		StorageEndpoint: job.StorageTarget.Endpoint,
		StoragePath:     job.StorageTarget.Path,
	}
	if job.Schedule != nil {
		resp.CronExpr = job.Schedule.CronExpr
	}
	// Decrypt credentials before sending to agent
	access, err := handlers.DecryptCredential(s.encryptionKey, job.StorageTarget.EncryptedAccessKey)
	if err == nil {
		resp.StorageAccessKey = access
	}
	secret, err := handlers.DecryptCredential(s.encryptionKey, job.StorageTarget.EncryptedSecretKey)
	if err == nil {
		resp.StorageSecretKey = secret
	}
	return resp, nil
}

// SendCommand dispatches a command to a connected agent by ID.
func (s *Server) SendCommand(agentID string, cmd *pb.ServerCommand) bool {
	s.mu.RLock()
	conn, ok := s.streams[agentID]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case conn.send <- cmd:
		return true
	default:
		return false
	}
}

func (s *Server) authenticateAgent(agentID, token string) (*models.Agent, error) {
	var agent models.Agent
	if err := s.db.Where("id = ? AND token = ?", agentID, token).First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}
