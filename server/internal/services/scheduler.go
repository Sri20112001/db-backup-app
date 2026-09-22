package services

import (
	"time"

	"github.com/backup-saas/server/internal/models"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Dispatcher sends a command to a connected agent. Satisfied by grpcserver.Server.
type Dispatcher interface {
	SendCommand(agentID string, cmd *pb.ServerCommand) bool
}

// Scheduler fires enabled jobs whose cron schedule is due by creating
// PENDING backup runs. Polling agents pick them up; connected gRPC agents
// also get a RUN_BACKUP command for backward compatibility.
type Scheduler struct {
	db       *gorm.DB
	dispatch Dispatcher
	interval time.Duration
	parser   cron.Parser
	stop     chan struct{}
	lastTick time.Time
}

func NewScheduler(db *gorm.DB, dispatch Dispatcher) *Scheduler {
	return &Scheduler{
		db:       db,
		dispatch: dispatch,
		interval: 30 * time.Second,
		parser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
		stop: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.lastTick = time.Now()
	go s.run()
}

func (s *Scheduler) Stop() { close(s.stop) }

func (s *Scheduler) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			s.tick(now)
		case <-s.stop:
			return
		}
	}
}

func (s *Scheduler) tick(now time.Time) {
	prev := s.lastTick
	s.lastTick = now

	var jobs []models.BackupJob
	if err := s.db.Preload("Schedule").
		Where("enabled = ?", true).Find(&jobs).Error; err != nil {
		log.Warn().Err(err).Msg("scheduler: list jobs failed")
		return
	}

	for i := range jobs {
		job := &jobs[i]
		if job.Schedule == nil || job.Schedule.CronExpr == "" {
			continue
		}
		if !s.due(job, prev, now) {
			continue
		}
		s.fire(job)
	}
}

// due reports whether the schedule crossed a firing boundary between prev and now.
func (s *Scheduler) due(job *models.BackupJob, prev, now time.Time) bool {
	sched, err := s.parser.Parse(job.Schedule.CronExpr)
	if err != nil {
		log.Warn().Str("job_id", job.ID.String()).
			Str("cron", job.Schedule.CronExpr).
			Msg("scheduler: invalid cron expression, skipping")
		return false
	}
	loc := time.UTC
	if job.Schedule.Timezone != "" {
		if l, err := time.LoadLocation(job.Schedule.Timezone); err == nil {
			loc = l
		}
	}
	next := sched.Next(prev.In(loc))
	return !next.After(now.In(loc))
}

// fire creates a PENDING run unless one is already in flight for the job,
// so an offline agent can't accumulate an unbounded backlog.
func (s *Scheduler) fire(job *models.BackupJob) {
	var inflight int64
	s.db.Model(&models.BackupRun{}).
		Where("backup_job_id = ? AND status IN ?", job.ID,
			[]models.BackupRunStatus{
				models.RunPending, models.RunRunning,
				models.RunUploading, models.RunVerifying,
			}).
		Count(&inflight)
	if inflight > 0 {
		log.Debug().Str("job_id", job.ID.String()).
			Msg("scheduler: previous run still in flight, skipping")
		return
	}

	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  job.OrganizationID,
		BackupJobID:     job.ID,
		AgentID:         job.AgentID,
		StorageTargetID: job.StorageTargetID,
		Status:          models.RunPending,
		SourceType:      string(job.SourceType),
	}
	if err := s.db.Create(&run).Error; err != nil {
		log.Warn().Err(err).Str("job_id", job.ID.String()).
			Msg("scheduler: create run failed")
		return
	}
	log.Info().Str("job_id", job.ID.String()).Str("run_id", run.ID.String()).
		Msg("scheduler: fired scheduled run")

	if s.dispatch != nil {
		s.dispatch.SendCommand(job.AgentID.String(), &pb.ServerCommand{
			Command: &pb.ServerCommand_RunBackup{
				RunBackup: &pb.RunBackupCommand{
					RunId: run.ID.String(),
					JobId: job.ID.String(),
				},
			},
		})
	}
}
