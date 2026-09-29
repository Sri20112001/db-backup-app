package api

import (
	"time"

	"github.com/backup-saas/server/internal/config"
	"github.com/backup-saas/server/internal/grpcserver"
	"github.com/backup-saas/server/internal/handlers"
	"github.com/backup-saas/server/internal/middleware"
	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/backup-saas/server/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config, grpcSrv *grpcserver.Server, hub *realtime.Hub) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/api/health"}}))
	r.Use(middleware.CORS(cfg.CORSOrigin))

	encKey := make([]byte, 32)
	copy(encKey, []byte(cfg.EncryptionKey))
	mailer := services.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)

	authH    := handlers.NewAuthHandler(db, cfg.JWTSecret, cfg.JWTRefreshSecret,
		time.Duration(cfg.AccessTokenTTLMinutes)*time.Minute,
		time.Duration(cfg.RefreshTokenTTLDays)*24*time.Hour)
	orgH     := handlers.NewOrgHandler(db)
	agentH   := handlers.NewAgentHandler(db, grpcSrv)
	machineH := handlers.NewMachineHandler(db)
	storageH := handlers.NewStorageHandler(db, encKey)
	jobH     := handlers.NewBackupJobHandler(db, grpcSrv)
	runH     := handlers.NewBackupRunHandler(db, grpcSrv, encKey, mailer)
	restoreH := handlers.NewRestoreHandler(db, grpcSrv, encKey)
	alertH   := handlers.NewAlertHandler(db)
	dashH    := handlers.NewDashboardHandler(db)
	userH    := handlers.NewUserHandler(db)

	preflightH := handlers.NewPreflightHandler(db)

	api := r.Group("/vaultguard/api")
	api.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(503, gin.H{"status": "degraded", "db": "unreachable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Realtime event bus (own JWT/agent handshake, not the JWT group).
	api.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

	// Public
	auth := api.Group("/auth")
	authLoginLimit := middleware.NewRateLimiter(10, time.Minute)
	auth.POST("/register", authLoginLimit.LimitAuth(), authH.Register)
	auth.POST("/login", authLoginLimit.LimitAuth(), authH.Login)
	auth.POST("/refresh", authH.Refresh)
	auth.POST("/logout", authH.Logout)

	// Agent self-registration (one-time key, no agent token yet)
	api.POST("/agents/register", agentH.Register)

	// All agent-authenticated endpoints share one middleware group so
	// protection is structural — a new handler added here is automatically
	// covered without needing its own inline auth check.
	workH := handlers.NewAgentWorkHandler(db, encKey)
	agentRoutes := api.Group("")
	agentRoutes.Use(middleware.AgentAuth(db))
	agentRoutes.PUT("/backup-runs/:id/status", runH.UpdateStatus)
	agentRoutes.POST("/backup-runs/:id/artifacts", runH.RegisterArtifact)
	agentRoutes.POST("/artifacts/:artifact_id/chunks", runH.RegisterChunk)
	agentRoutes.PUT("/restores/:id/status", restoreH.UpdateStatus)
	agentRoutes.GET("/agent/runs", workH.PendingRuns)
	agentRoutes.POST("/agent/runs/:id/claim", workH.ClaimRun)
	agentRoutes.GET("/agent/restores", workH.PendingRestores)
	agentRoutes.POST("/agent/restores/:id/claim", workH.ClaimRestore)

	// JWT-authenticated routes
	authed := api.Group("")
	authed.Use(middleware.Auth(cfg.JWTSecret))
	authed.Use(middleware.Audit(db))

	authed.GET("/me", userH.Me)
	authed.POST("/auth/change-password", authH.ChangePassword)
	authed.GET("/organizations", orgH.List)
	authed.POST("/organizations", orgH.Create)

	org := authed.Group("/organizations/:org_id")
	org.Use(middleware.OrgContext(db))

	org.GET("", orgH.Get)

	// Members
	org.GET("/members", userH.ListMembers)
	org.POST("/members", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), userH.InviteMember)
	org.PUT("/members/:user_id/role", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), userH.UpdateRole)
	org.DELETE("/members/:user_id", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), userH.RemoveMember)

	// Agents
	org.GET("/agents", agentH.List)
	org.POST("/agents/token", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), agentH.GenerateRegistrationToken)
	org.GET("/agents/:id", agentH.Get)
	org.POST("/agents/:id/rotate-token", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), agentH.RotateToken)
	org.DELETE("/agents/:id", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), agentH.Delete)

	// Machines
	org.GET("/machines", machineH.List)
	org.GET("/machines/:id", machineH.Get)

	// Storage targets
	org.GET("/storage-targets", storageH.List)
	org.POST("/storage-targets", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), storageH.Create)
	org.DELETE("/storage-targets/:id", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), storageH.Delete)

	// Backup jobs
	org.GET("/backup-jobs", jobH.List)
	org.POST("/backup-jobs", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), jobH.Create)
	org.GET("/backup-jobs/:id", jobH.Get)
	org.PUT("/backup-jobs/:id", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), jobH.Update)
	org.DELETE("/backup-jobs/:id", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), jobH.Delete)
	org.POST("/backup-jobs/:id/run", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), jobH.RunNow)
	org.POST("/backup-jobs/:id/enable", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), jobH.Enable)
	org.POST("/backup-jobs/:id/disable", middleware.RequireRole(models.RoleOwner, models.RoleAdmin), jobH.Disable)
	org.GET("/backup-jobs/:id/preflight", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), preflightH.Preflight)

	// Backup runs
	org.GET("/backup-runs", runH.List)
	org.GET("/backup-runs/:id", runH.Get)
	org.POST("/backup-runs/:id/cancel", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), runH.Cancel)
	org.POST("/backup-runs/:id/verify", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), runH.Verify)
	org.GET("/backup-runs/:id/artifacts", runH.ListArtifacts)

	// Restores
	org.GET("/restores", restoreH.List)
	org.POST("/restores", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), restoreH.Create)
	org.GET("/restores/:id", restoreH.Get)

	// Alerts
	org.GET("/alerts", alertH.List)
	org.PUT("/alerts/read-all", alertH.MarkAllRead)
	org.PUT("/alerts/:id/read", alertH.MarkRead)

	// Dashboard
	org.GET("/dashboard", dashH.Overview)
	org.GET("/dashboard/health", dashH.BackupHealth)

	return r
}
