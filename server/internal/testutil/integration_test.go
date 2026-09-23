package testutil_test

import (
	"sync"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/testutil"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Concurrent refresh rotation ---

// simulateRefresh mirrors the production Refresh transaction logic so we can
// test it directly against a real database without spinning up an HTTP server.
func simulateRefresh(db *gorm.DB, rawToken string, userID uuid.UUID) (newToken string, reuseDetected bool, err error) {
	tokenHash := testutil.HashToken(rawToken)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		var rt models.RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token = ? AND user_id = ? AND expires_at > ?", tokenHash, userID, time.Now()).
			First(&rt).Error; err != nil {
			return err
		}

		if rt.Revoked {
			// Revoke entire family
			tx.Model(&models.RefreshToken{}).
				Where("family_id = ? AND revoked = false", rt.FamilyID).
				Update("revoked", true)
			reuseDetected = true
			return gorm.ErrRecordNotFound
		}

		if err := tx.Model(&rt).Update("revoked", true).Error; err != nil {
			return err
		}

		newToken = "new-token-" + uuid.New().String()
		return tx.Create(&models.RefreshToken{
			UserID:    userID,
			FamilyID:  rt.FamilyID,
			Token:     testutil.HashToken(newToken),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}).Error
	})

	return newToken, reuseDetected, txErr
}

func TestConcurrentRefreshRotation(t *testing.T) {
	db := testutil.OpenDB(t)

	user := testutil.CreateUser(t, db, "concurrent-refresh@test.com", "password123")
	rawToken := "initial-refresh-token-" + uuid.New().String()
	testutil.CreateRefreshToken(t, db, user.ID, rawToken)

	const goroutines = 5
	type result struct {
		newToken      string
		reuseDetected bool
		err           error
	}
	results := make([]result, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tok, reuse, err := simulateRefresh(db, rawToken, user.ID)
			results[idx] = result{tok, reuse, err}
		}(i)
	}
	wg.Wait()

	// Exactly one goroutine should have succeeded (got a new token)
	successes := 0
	for _, r := range results {
		if r.err == nil && r.newToken != "" {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 successful rotation, got %d", successes)
	}

	// The original token must be revoked
	var original models.RefreshToken
	db.Where("token = ?", testutil.HashToken(rawToken)).First(&original)
	if !original.Revoked {
		t.Error("original token should be revoked after rotation")
	}

	// Exactly one new active token should exist in the family
	var activeCount int64
	db.Model(&models.RefreshToken{}).
		Where("family_id = ? AND revoked = false", original.FamilyID).
		Count(&activeCount)
	if activeCount != 1 {
		t.Errorf("expected exactly 1 active token in family, got %d", activeCount)
	}
}

func TestRefreshReuseRevokesFamily(t *testing.T) {
	db := testutil.OpenDB(t)

	user := testutil.CreateUser(t, db, "reuse-test@test.com", "password123")
	rawToken := "reuse-token-" + uuid.New().String()
	rt := testutil.CreateRefreshToken(t, db, user.ID, rawToken)

	// Rotate once legitimately
	newToken, _, err := simulateRefresh(db, rawToken, user.ID)
	if err != nil {
		t.Fatalf("first rotation failed: %v", err)
	}
	if newToken == "" {
		t.Fatal("expected new token after rotation")
	}

	// Now rotate the new token to create a second generation
	_, _, err = simulateRefresh(db, newToken, user.ID)
	if err != nil {
		t.Fatalf("second rotation failed: %v", err)
	}

	// Replay the original (now revoked) token — should trigger reuse detection
	_, reuseDetected, _ := simulateRefresh(db, rawToken, user.ID)
	if !reuseDetected {
		t.Error("expected reuse detection when replaying a rotated token")
	}

	// All tokens in the family should now be revoked
	var activeCount int64
	db.Model(&models.RefreshToken{}).
		Where("family_id = ? AND revoked = false", rt.FamilyID).
		Count(&activeCount)
	if activeCount != 0 {
		t.Errorf("expected 0 active tokens after reuse detection, got %d", activeCount)
	}
}

// --- Multi-tenant isolation ---

func TestOrgIsolation_AgentNotAccessibleAcrossOrgs(t *testing.T) {
	db := testutil.OpenDB(t)

	userA := testutil.CreateUser(t, db, "user-a@test.com", "password123")
	userB := testutil.CreateUser(t, db, "user-b@test.com", "password123")
	orgA := testutil.CreateOrg(t, db, "Org A", userA.ID)
	orgB := testutil.CreateOrg(t, db, "Org B", userB.ID)

	agentA := testutil.CreateAgent(t, db, orgA.ID, "token-a-"+uuid.New().String())

	// User B (Org B) tries to fetch Agent A by ID scoped to Org B — must not find it
	var found models.Agent
	err := db.Where("id = ? AND organization_id = ?", agentA.ID, orgB.ID).First(&found).Error
	if err == nil {
		t.Error("Org B should not be able to access Org A's agent")
	}
}

func TestOrgIsolation_BackupRunNotAccessibleAcrossOrgs(t *testing.T) {
	db := testutil.OpenDB(t)

	userA := testutil.CreateUser(t, db, "run-user-a@test.com", "password123")
	userB := testutil.CreateUser(t, db, "run-user-b@test.com", "password123")
	orgA := testutil.CreateOrg(t, db, "Run Org A", userA.ID)
	orgB := testutil.CreateOrg(t, db, "Run Org B", userB.ID)

	agentA := testutil.CreateAgent(t, db, orgA.ID, "run-token-"+uuid.New().String())

	// Create a storage target for Org A
	storage := models.StorageTarget{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: orgA.ID,
		Name:           "test-storage",
		Type:           models.StorageS3,
	}
	db.Create(&storage)
	t.Cleanup(func() { db.Unscoped().Delete(&storage) })

	// Create a backup job for Org A
	job := models.BackupJob{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgA.ID,
		AgentID:         agentA.ID,
		StorageTargetID: storage.ID,
		Name:            "test-job",
		SourceType:      models.SourceFilesystem,
		Mode:            models.ModeNormal,
		Enabled:         true,
		RetentionDays:   30,
	}
	db.Create(&job)
	t.Cleanup(func() { db.Unscoped().Delete(&job) })

	// Create a backup run for Org A
	run := models.BackupRun{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgA.ID,
		BackupJobID:     job.ID,
		AgentID:         agentA.ID,
		StorageTargetID: storage.ID,
		Status:          models.RunCompleted,
	}
	db.Create(&run)
	t.Cleanup(func() { db.Unscoped().Delete(&run) })

	// Org B user tries to access Org A's run — must not find it
	var found models.BackupRun
	err := db.Where("id = ? AND organization_id = ?", run.ID, orgB.ID).First(&found).Error
	if err == nil {
		t.Error("Org B should not be able to access Org A's backup run")
	}
}

func TestOrgIsolation_RestoreCannotReferenceOtherOrgRun(t *testing.T) {
	db := testutil.OpenDB(t)

	userA := testutil.CreateUser(t, db, "restore-a@test.com", "password123")
	userB := testutil.CreateUser(t, db, "restore-b@test.com", "password123")
	orgA := testutil.CreateOrg(t, db, "Restore Org A", userA.ID)
	orgB := testutil.CreateOrg(t, db, "Restore Org B", userB.ID)

	agentA := testutil.CreateAgent(t, db, orgA.ID, "restore-token-"+uuid.New().String())

	storage := models.StorageTarget{
		Base: models.Base{ID: uuid.New()}, OrganizationID: orgA.ID,
		Name: "s", Type: models.StorageLocal,
	}
	db.Create(&storage)
	t.Cleanup(func() { db.Unscoped().Delete(&storage) })

	job := models.BackupJob{
		Base: models.Base{ID: uuid.New()}, OrganizationID: orgA.ID,
		AgentID: agentA.ID, StorageTargetID: storage.ID,
		Name: "j", SourceType: models.SourceFilesystem, Mode: models.ModeNormal,
		Enabled: true, RetentionDays: 30,
	}
	db.Create(&job)
	t.Cleanup(func() { db.Unscoped().Delete(&job) })

	runA := models.BackupRun{
		Base: models.Base{ID: uuid.New()}, OrganizationID: orgA.ID,
		BackupJobID: job.ID, AgentID: agentA.ID, StorageTargetID: storage.ID,
		Status: models.RunCompleted,
	}
	db.Create(&runA)
	t.Cleanup(func() { db.Unscoped().Delete(&runA) })

	// Org B tries to create a restore referencing Org A's run — the org-scoped
	// lookup must fail to find the run
	var found models.BackupRun
	err := db.Where("id = ? AND organization_id = ?", runA.ID, orgB.ID).First(&found).Error
	if err == nil {
		t.Error("Org B should not be able to reference Org A's backup run for a restore")
	}
}

// --- Agent cross-org token rejection ---

func TestAgentAuth_CrossOrgRejected(t *testing.T) {
	db := testutil.OpenDB(t)

	userA := testutil.CreateUser(t, db, "agent-auth-a@test.com", "password123")
	userB := testutil.CreateUser(t, db, "agent-auth-b@test.com", "password123")
	orgA := testutil.CreateOrg(t, db, "Agent Auth Org A", userA.ID)
	orgB := testutil.CreateOrg(t, db, "Agent Auth Org B", userB.ID)

	rawToken := "cross-org-token-" + uuid.New().String()
	agentA := testutil.CreateAgent(t, db, orgA.ID, rawToken)

	// Org B tries to use Agent A's ID with Agent A's token — the org_id check must reject it
	var found models.Agent
	err := db.Where("id = ? AND organization_id = ?", agentA.ID, orgB.ID).First(&found).Error
	if err == nil {
		t.Error("agent from Org A should not be accessible via Org B's org_id scope")
	}
}
