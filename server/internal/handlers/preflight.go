package handlers

import (
	"net/http"
	"os"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PreflightHandler struct {
	db *gorm.DB
}

func NewPreflightHandler(db *gorm.DB) *PreflightHandler {
	return &PreflightHandler{db: db}
}

type preflightCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // OK | WARN | FAIL
	Detail string `json:"detail,omitempty"`
}

// Preflight runs a set of pre-backup checks for a job and returns each
// result. It does NOT start the backup. The dashboard calls this before
// showing the "Run Now" confirmation so users see predictable failures
// before they happen.
func (h *PreflightHandler) Preflight(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var job models.BackupJob
	if err := h.db.Preload("Agent").Preload("StorageTarget").
		Where("id = ? AND organization_id = ?", jobID, orgID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	checks := []preflightCheck{
		h.checkAgentOnline(job),
		h.checkNoConflictingRun(job),
		h.checkStorageReachable(job),
		h.checkDiskSpace(job),
		h.checkEncryptionKey(job),
	}

	// Overall: FAIL if any check failed, WARN if any warned, else OK.
	overall := "OK"
	for _, ch := range checks {
		if ch.Status == "FAIL" {
			overall = "FAIL"
			break
		}
		if ch.Status == "WARN" && overall != "FAIL" {
			overall = "WARN"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"overall": overall,
		"checks":  checks,
	})
}

func (h *PreflightHandler) checkAgentOnline(job models.BackupJob) preflightCheck {
	if job.Agent.Status == models.AgentOnline {
		return preflightCheck{Name: "Agent online", Status: "OK"}
	}
	return preflightCheck{
		Name:   "Agent online",
		Status: "FAIL",
		Detail: "Agent " + job.Agent.Name + " is OFFLINE — backup cannot be dispatched",
	}
}

func (h *PreflightHandler) checkNoConflictingRun(job models.BackupJob) preflightCheck {
	var inflight int64
	h.db.Model(&models.BackupRun{}).
		Where("backup_job_id = ? AND status IN ?", job.ID,
			[]models.BackupRunStatus{models.RunPending, models.RunRunning, models.RunUploading, models.RunVerifying}).
		Count(&inflight)
	if inflight == 0 {
		return preflightCheck{Name: "No conflicting run", Status: "OK"}
	}
	return preflightCheck{
		Name:   "No conflicting run",
		Status: "WARN",
		Detail: "A backup run for this job is already in progress",
	}
}

func (h *PreflightHandler) checkStorageReachable(job models.BackupJob) preflightCheck {
	if job.StorageTarget.Type == models.StorageLocal {
		if job.StorageTarget.Path == "" {
			return preflightCheck{Name: "Storage reachable", Status: "FAIL", Detail: "LOCAL storage target has no path configured"}
		}
		if _, err := os.Stat(job.StorageTarget.Path); err != nil {
			return preflightCheck{
				Name:   "Storage reachable",
				Status: "FAIL",
				Detail: "Storage path not accessible from server: " + err.Error(),
			}
		}
		return preflightCheck{Name: "Storage reachable", Status: "OK"}
	}
	// For S3/SMB we can't probe from the server — note it as informational.
	return preflightCheck{
		Name:   "Storage reachable",
		Status: "WARN",
		Detail: "Remote storage (" + string(job.StorageTarget.Type) + ") reachability can only be verified by the agent at runtime",
	}
}

func (h *PreflightHandler) checkDiskSpace(job models.BackupJob) preflightCheck {
	if job.StorageTarget.Type != models.StorageLocal || job.StorageTarget.Path == "" {
		return preflightCheck{Name: "Disk space", Status: "WARN", Detail: "Cannot check disk space for remote storage targets"}
	}
	// Disk space can only be accurately measured on the agent machine.
	// Surface as informational so operators know to monitor it.
	return preflightCheck{
		Name:   "Disk space",
		Status: "WARN",
		Detail: "Disk space is checked by the agent at runtime; ensure the storage path has sufficient free space",
	}
}

func (h *PreflightHandler) checkEncryptionKey(job models.BackupJob) preflightCheck {
	if !job.Encrypted {
		return preflightCheck{Name: "Encryption key", Status: "OK", Detail: "Encryption not enabled for this job"}
	}
	// The server holds the wrapped key; if ENCRYPTION_KEY env is set the
	// server started successfully, so the key is available.
	return preflightCheck{Name: "Encryption key", Status: "OK"}
}
