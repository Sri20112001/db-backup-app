package handlers

import (
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/google/uuid"
)

// The connection's own status is a cached snapshot written when the server
// dialed the database; it is never invalidated when the agent goes away. The
// DTO therefore also carries the agent lifecycle so the UI can stop showing
// "Connected" for a pending/offline/revoked agent.
func TestToConnectionDTOAgentLifecycle(t *testing.T) {
	hash := "hash"
	now := time.Now()

	pending := models.DatabaseConnection{
		Base:  models.Base{ID: uuid.New()},
		Agent: models.Agent{Base: models.Base{ID: uuid.New()}}, // no TokenHash
	}
	if got := toConnectionDTO(pending)["agent_lifecycle"]; got != models.LifecyclePending {
		t.Fatalf("pending agent lifecycle = %v, want %s", got, models.LifecyclePending)
	}

	online := models.DatabaseConnection{
		Base: models.Base{ID: uuid.New()},
		Agent: models.Agent{
			Base:       models.Base{ID: uuid.New()},
			TokenHash:  &hash,
			LastSeenAt: &now,
		},
	}
	if got := toConnectionDTO(online)["agent_lifecycle"]; got != models.LifecycleOnline {
		t.Fatalf("online agent lifecycle = %v, want %s", got, models.LifecycleOnline)
	}

	offline := now.Add(-10 * time.Minute)
	stale := models.DatabaseConnection{
		Base: models.Base{ID: uuid.New()},
		Agent: models.Agent{
			Base:       models.Base{ID: uuid.New()},
			TokenHash:  &hash,
			LastSeenAt: &offline,
		},
	}
	if got := toConnectionDTO(stale)["agent_lifecycle"]; got != models.LifecycleOffline {
		t.Fatalf("stale agent lifecycle = %v, want %s", got, models.LifecycleOffline)
	}

	// Agent relation not preloaded: a zero-value Agent would read as PENDING,
	// so the key must be omitted entirely instead of lying.
	unloaded := models.DatabaseConnection{Base: models.Base{ID: uuid.New()}}
	if _, ok := toConnectionDTO(unloaded)["agent_lifecycle"]; ok {
		t.Fatal("agent_lifecycle must be omitted when the Agent relation is not loaded")
	}
}
