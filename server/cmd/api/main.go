package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/backup-saas/server/internal/api"
	"github.com/backup-saas/server/internal/config"
	"github.com/backup-saas/server/internal/db"
	"github.com/backup-saas/server/internal/grpcserver"
	"github.com/backup-saas/server/internal/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	config.LoadEnv()
	cfg := config.AppEnv

	log.Info().
		Str("host", cfg.DBHost).
		Str("user", cfg.DBUser).
		Str("dbname", cfg.DBName).
		Int("port", cfg.DBPort).
		Str("sslmode", cfg.DBSSLMode).
		Msg("connecting to database")

	database, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	if err := db.Migrate(database); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	encKey := make([]byte, 32)
	copy(encKey, []byte(cfg.EncryptionKey))

	monitor := services.NewHealthMonitor(database)
	monitor.Start()
	defer monitor.Stop()

	grpcSrv := grpcserver.NewServer(database, encKey)
	go func() {
		if err := grpcSrv.Start(":" + cfg.GRPCPort); err != nil {
			log.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	scheduler := services.NewScheduler(database, grpcSrv)
	scheduler.Start()
	defer scheduler.Stop()

	router := api.NewRouter(database, cfg, grpcSrv)

	go func() {
		log.Info().Str("port", cfg.Port).Msg("HTTP server listening")
		if err := router.Run(":" + cfg.Port); err != nil {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down")
}
