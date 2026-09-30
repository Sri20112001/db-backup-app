package db

import (
	"os"
	"testing"

	"github.com/backup-saas/server/internal/models"
)

// TestMigrateUpSeedsRegions proves a fresh database comes out of the
// versioned chain with the S3 region reference data present — the region
// picker depends on it. DB-gated like the rest of the suite.
func TestMigrateUpSeedsRegions(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := Connect(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer sqlDB.Close()

	var count int64
	database.Model(&models.S3Region{}).Where("active = ?", true).Count(&count)
	if count < 25 {
		t.Fatalf("expected seeded regions (>= 25), got %d", count)
	}
	var auto models.S3Region
	if err := database.Where("code = ?", "auto").First(&auto).Error; err != nil {
		t.Fatalf("missing 'auto' S3-compatible preset: %v", err)
	}
}
