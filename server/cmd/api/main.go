package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/backup-saas/server/internal/api"
	"github.com/backup-saas/server/internal/config"
	"github.com/backup-saas/server/internal/db"
	"github.com/backup-saas/server/internal/grpcserver"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/backup-saas/server/internal/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	// Versioned migrations (embedded SQL). If a previous run died mid-migration
	// the version row is left dirty and the server refuses to guess — clear it
	// with the migrate CLI (`… force <version>`) after confirming which half
	// of that version actually applied.
	if err := db.MigrateUp(cfg.DatabaseURL); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	encKey := make([]byte, 32)
	copy(encKey, []byte(cfg.EncryptionKey))

	if cfg.EncryptionKey == "" {
		log.Warn().Msg("ENCRYPTION_KEY is not set — storage credentials will be stored in plaintext")
	}

	mailer := services.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	if !mailer.Enabled() {
		log.Warn().Msg("SMTP_HOST is not set — alert emails are disabled (alerts still recorded)")
	}

	monitor := services.NewHealthMonitor(database, mailer)
	monitor.Start()
	defer monitor.Stop()

	grpcSrv := grpcserver.NewServer(database, encKey, mailer)
	go func() {
		if err := grpcSrv.Start(":" + cfg.GRPCPort); err != nil {
			log.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	// Realtime event bus for dashboard/agent sockets.
	realtime.DefaultHub = realtime.NewHub(database, cfg.JWTSecret, cfg.CORSOrigin)

	router := api.NewRouter(database, cfg, grpcSrv, realtime.DefaultHub)

	// Fires PENDING runs for due cron schedules; polling agents pick them up.
	scheduler := services.NewScheduler(database, grpcSrv)
	scheduler.Start()
	defer scheduler.Stop()

	go func() {
		addr := ":" + cfg.Port
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			log.Info().Str("port", cfg.Port).Msg("HTTPS server listening")
			if err := router.RunTLS(addr, cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
				log.Fatal().Err(err).Msg("HTTPS server failed")
			}
			return
		}
		log.Info().Str("port", cfg.Port).Msg("HTTP server listening (TLS off — set TLS_CERT_FILE/TLS_KEY_FILE for HTTPS)")
		if err := router.Run(addr); err != nil {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down")
}
