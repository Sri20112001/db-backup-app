package services

import (
	"time"

	"github.com/backup-saas/server/internal/models"
	pb "github.com/backup-saas/server/proto/agentpb"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := s.db.Preload("Schedule").Preload("Policy").
		Where("enabled = ?", true).Find(&jobs).Error; err != nil {
		log.Warn().Err(err).Msg("scheduler: list jobs failed")
		return
	}

	for i := range jobs {
		job := &jobs[i]
		// Effective schedule: policy cron wins when attached (and can also
		// disable scheduling via policy), else the job's own schedule row.
		// Policy-less jobs behave exactly as before (zero migration).
		eff := models.EffectivePolicy(job)
		if !eff.Enabled {
			continue
		}
		// Retries are schedule-independent: a manual job's failed run
		// refires too. Disabled policies/jobs never retry.
		if err := s.maybeFire(job, eff, prev, now); err != nil {
			log.Warn().Err(err).Str("job_id", job.ID.String()).
				Msg("scheduler: retry check failed")
		}
		if eff.CronExpr == "" {
			continue
		}
		boundary, ok := s.boundaryFor(eff.CronExpr, eff.Timezone, job.ID, prev, now)
		if !ok {
			continue
		}
		s.fire(job, boundary)
	}
}

// due reports whether the schedule crossed a firing boundary between prev and now.
func (s *Scheduler) due(job *models.BackupJob, prev, now time.Time) bool {
	_, ok := s.boundary(job, prev, now)
	return ok
}

// boundary returns the exact cron instant crossed between prev and now.
// The instant doubles as the run's idempotency key (see fire).
// Kept for the job-inline path and existing tests; tick() resolves the
// effective schedule first and calls boundaryFor.
func (s *Scheduler) boundary(job *models.BackupJob, prev, now time.Time) (time.Time, bool) {
	if job.Schedule == nil {
		return time.Time{}, false
	}
	return s.boundaryFor(job.Schedule.CronExpr, job.Schedule.Timezone, job.ID, prev, now)
}

// boundaryFor is the schedule-agnostic core: jobs resolve their effective
// cron/timezone (policy or inline) before calling it, so the firing math
// lives in exactly one place.
func (s *Scheduler) boundaryFor(cronExpr, timezone string, jobID uuid.UUID, prev, now time.Time) (time.Time, bool) {
	if cronExpr == "" {
		return time.Time{}, false
	}
	sched, err := s.parser.Parse(cronExpr)
	if err != nil {
		log.Warn().Str("job_id", jobID.String()).
			Str("cron", cronExpr).
			Msg("scheduler: invalid cron expression, skipping")
		return time.Time{}, false
	}
	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}
	next := sched.Next(prev.In(loc))
	if next.After(now.In(loc)) {
		return time.Time{}, false
	}
	return next, true
}

// maybeFire refires the latest FAILED run as a NEW linked PENDING run when
// the effective policy allows retries. Terminal states never transition out,
// so a retry is a fresh run carrying RetryOfRunID/RetryAttempt (observable
// in history and the run detail drawer). All conditions must hold:
//   - policy enabled with MaxRetries > 0
//   - nothing currently in flight (same guard as fire)
//   - the latest run is FAILED with attempts remaining
//   - the failure is older than the retry delay (no hot loops)
//   - the failed run is still the latest (fresh scheduled work wins)
//
// Like fire, dispatch is best-effort: polling agents pick up PENDING runs
// regardless of the gRPC nudge.
func (s *Scheduler) maybeFire(job *models.BackupJob, eff models.ResolvedPolicy, prev, now time.Time) error {
	if !eff.Enabled || eff.MaxRetries <= 0 {
		return nil
	}
	var inflight int64
	if err := s.db.Model(&models.BackupRun{}).
		Where("backup_job_id = ? AND status IN ?", job.ID,
			[]models.BackupRunStatus{
				models.RunPending, models.RunRunning,
				models.RunUploading, models.RunVerifying,
			}).
		Count(&inflight).Error; err != nil {
		return err
	}
	if inflight > 0 {
		return nil
	}
	var latest models.BackupRun
	if err := s.db.Where("backup_job_id = ?", job.ID).
		Order("created_at DESC").First(&latest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if latest.Status != models.RunFailed {
		return nil
	}
	if latest.RetryAttempt >= eff.MaxRetries {
		return nil
	}
	delay := eff.RetryDelaySeconds
	if delay <= 0 {
		delay = 300
	}
	if latest.CompletedAt == nil || now.Sub(*latest.CompletedAt) < time.Duration(delay)*time.Second {
		return nil
	}
	retryID := latest.ID
	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  job.OrganizationID,
		BackupJobID:     job.ID,
		AgentID:         job.AgentID,
		StorageTargetID: job.StorageTargetID,
		Status:          models.RunPending,
		SourceType:      string(job.SourceType),
		// ScheduledFor stays NULL (like RunNow): retries are ad-hoc refires,
		// not cron boundaries, so they can never collide on the idempotency
		// unique index.
		RetryOfRunID: &retryID,
		RetryAttempt: latest.RetryAttempt + 1,
	}
	if err := s.db.Create(&run).Error; err != nil {
		return err
	}
	log.Info().Str("job_id", job.ID.String()).Str("run_id", run.ID.String()).
		Int("attempt", run.RetryAttempt).Msg("scheduler: fired retry run")
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
	return nil
}

// fire creates a PENDING run unless one is already in flight for the job,
// so an offline agent can't accumulate an unbounded backlog.
//
// Race safety: the insert carries (backup_job_id, scheduled_for) with
// ON CONFLICT DO NOTHING against idx_backup_runs_job_scheduled. Two ticks
// racing — multi-instance schedulers, restarts, overlapping windows —
// resolve to exactly one run; the loser sees RowsAffected == 0.
func (s *Scheduler) fire(job *models.BackupJob, scheduledFor time.Time) {
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

	boundary := scheduledFor.UTC()
	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  job.OrganizationID,
		BackupJobID:     job.ID,
		AgentID:         job.AgentID,
		StorageTargetID: job.StorageTargetID,
		Status:          models.RunPending,
		SourceType:      string(job.SourceType),
		ScheduledFor:    &boundary,
	}
	res := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "backup_job_id"},
			{Name: "scheduled_for"},
		},
		DoNothing: true,
	}).Create(&run)
	if res.Error != nil {
		log.Warn().Err(res.Error).Str("job_id", job.ID.String()).
			Msg("scheduler: create run failed")
		return
	}
	if res.RowsAffected == 0 {
		log.Debug().Str("job_id", job.ID.String()).
			Msg("scheduler: boundary already fired (lost race), skipping")
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
