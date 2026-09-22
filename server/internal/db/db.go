package db

import (
	"fmt"

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

	if err := migrateLegacyUsers(db); err != nil {
		return err
	}

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

// migrateLegacyUsers bridges the pre-existing users table (id TEXT,
// username/password columns) to the current schema before AutoMigrate
// runs. Fresh databases skip this entirely.
func migrateLegacyUsers(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.User{}) {
		return nil
	}

	if !db.Migrator().HasColumn(&models.User{}, "password_hash") {
		log.Info().Msg("migrating legacy users table: adding password_hash column")
		if err := db.Exec("ALTER TABLE users ADD COLUMN password_hash TEXT").Error; err != nil {
			return fmt.Errorf("add users.password_hash column: %w", err)
		}
		// Preserve existing logins by carrying over the old bcrypt hashes.
		if err := db.Exec("UPDATE users SET password_hash = COALESCE(password, '') WHERE password_hash IS NULL").Error; err != nil {
			return fmt.Errorf("backfill users.password_hash: %w", err)
		}
	} else if err := db.Exec("UPDATE users SET password_hash = '' WHERE password_hash IS NULL").Error; err != nil {
		return fmt.Errorf("backfill null users.password_hash: %w", err)
	}

	var dataType string
	if err := db.Raw("SELECT data_type FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'id'").Scan(&dataType).Error; err != nil {
		return fmt.Errorf("inspect users.id type: %w", err)
	}
	if dataType == "text" {
		log.Info().Msg("migrating legacy users table: converting id to uuid")
		if err := db.Exec("ALTER TABLE users ALTER COLUMN id TYPE uuid USING id::uuid").Error; err != nil {
			return fmt.Errorf("convert users.id to uuid: %w", err)
		}
	}
	return nil
}
