package handlers

import (
	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
)

// asArray normalizes a nil slice to an empty one so JSON encodes [] instead
// of null. The frontend types every list as an array (e.g. Agent[]) and
// crashes on null (e.g. agents.length). GORM leaves slices nil when no rows
// match, so wrap every slice at the JSON boundary.
func asArray[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// publishRun broadcasts a backup-run change to dashboards of the run's org.
// Nil-hub safe (see realtime.DefaultHub).
func publishRun(run models.BackupRun, status models.BackupRunStatus, bytesRead, bytesCompressed, bytesUploaded int64, checksum, errorMessage, storagePath string) {
	realtime.Publish(realtime.DefaultHub, run.OrganizationID.String(), realtime.Event{
		Type: realtime.TypeRun,
		Payload: map[string]interface{}{
			"id":               run.ID.String(),
			"backup_job_id":    run.BackupJobID.String(),
			"agent_id":         run.AgentID.String(),
			"organization_id":  run.OrganizationID.String(),
			"status":           string(status),
			"bytes_read":       bytesRead,
			"bytes_compressed": bytesCompressed,
			"bytes_uploaded":   bytesUploaded,
			"checksum":         checksum,
			"error_message":    errorMessage,
			"storage_path":     storagePath,
		},
	})
}

// publishEntity broadcasts a collection-level change (jobs/restores/agents)
// so other operators' lists refresh.
func publishEntity(orgID string, eventType, action, id string) {
	realtime.Publish(realtime.DefaultHub, orgID, realtime.Event{
		Type:    eventType,
		Payload: map[string]interface{}{"action": action, "id": id},
	})
}

// publishAlert broadcasts a freshly created alert row.
func publishAlert(alert models.Alert) {
	realtime.Publish(realtime.DefaultHub, alert.OrganizationID.String(), realtime.Event{
		Type:    realtime.TypeAlert,
		Payload: alert,
	})
}

// publishPresence broadcasts an agent ONLINE/OFFLINE flip.
func publishPresence(agent models.Agent) {
	realtime.Publish(realtime.DefaultHub, agent.OrganizationID.String(), realtime.Event{
		Type: realtime.TypePresence,
		Payload: map[string]interface{}{
			"id":     agent.ID.String(),
			"name":   agent.Name,
			"status": string(agent.Status),
		},
	})
}
