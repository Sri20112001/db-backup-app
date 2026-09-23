package db

import (
	"github.com/backup-saas/server/internal/models"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Info().Msg("running database migrations")
	return db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.OrganizationMember{},
		&models.Agent{},
		&models.Machine{},
		&models.StorageTarget{},
		&models.BackupJob{},
		&models.BackupSchedule{},
		&models.BackupRun{},
		&models.BackupArtifact{},
		&models.BackupChunk{},
		&models.RestoreJob{},
		&models.Alert{},
		&models.AuditLog{},
		&models.RefreshToken{},
	)
}
