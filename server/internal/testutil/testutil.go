// Package testutil provides helpers for PostgreSQL integration tests.
// Tests using OpenDB are skipped automatically when DATABASE_URL is not set.
package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/backup-saas/server/internal/db"
	"github.com/backup-saas/server/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// OpenDB connects to the test database and runs AutoMigrate.
// Skips the test if DATABASE_URL is not set.
func OpenDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	database, err := db.Connect(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

// HashToken mirrors the production hashToken function.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CreateUser inserts a user with the given email and password.
func CreateUser(t *testing.T, database *gorm.DB, email, password string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{
		Base:         models.Base{ID: uuid.New()},
		Email:        email + "-" + uuid.New().String()[:8], // unique per test run
		PasswordHash: string(hash),
		Name:         "Test User",
	}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&user) })
	return user
}

// CreateOrg inserts an org and makes the user its OWNER.
func CreateOrg(t *testing.T, database *gorm.DB, name string, ownerID uuid.UUID) models.Organization {
	t.Helper()
	org := models.Organization{
		Base:      models.Base{ID: uuid.New()},
		Name:      name,
		Slug:      "test-" + uuid.New().String()[:8],
		UserLimit: 10,
	}
	if err := database.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	database.Create(&models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         ownerID,
		Role:           models.RoleOwner,
	})
	t.Cleanup(func() {
		database.Unscoped().Where("organization_id = ?", org.ID).Delete(&models.OrganizationMember{})
		database.Unscoped().Delete(&org)
	})
	return org
}

// CreateAgent inserts an agent with a known raw token.
func CreateAgent(t *testing.T, database *gorm.DB, orgID uuid.UUID, rawToken string) models.Agent {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(rawToken), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash agent token: %v", err)
	}
	agent := models.Agent{
		Base:           models.Base{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           "test-agent-" + uuid.New().String()[:8],
		TokenHash:      string(hash),
		Status:         models.AgentOnline,
	}
	if err := database.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&agent) })
	return agent
}

// CreateRefreshToken inserts a hashed refresh token for a user.
func CreateRefreshToken(t *testing.T, database *gorm.DB, userID uuid.UUID, rawToken string) models.RefreshToken {
	t.Helper()
	rt := models.RefreshToken{
		UserID:    userID,
		FamilyID:  uuid.New(),
		Token:     HashToken(rawToken),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := database.Create(&rt).Error; err != nil {
		t.Fatalf("create refresh token: %v", err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&rt) })
	return rt
}
