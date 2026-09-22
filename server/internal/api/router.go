package api

import (
	"github.com/backup-saas/server/internal/config"
	"github.com/backup-saas/server/internal/grpcserver"
	"github.com/backup-saas/server/internal/handlers"
	"github.com/backup-saas/server/internal/middleware"
	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.ENV, grpcSrv *grpcserver.Server) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/api/health"}}))
	r.Use(middleware.CORS(cfg.CORSOrigin))

	encKey := make([]byte, 32)
	copy(encKey, []byte(cfg.EncryptionKey))

	authH    := handlers.NewAuthHandler(db, cfg.JWTSecretKey, cfg.JWTRefreshSecret)
	orgH     := handlers.NewOrgHandler(db)
	agentH   := handlers.NewAgentHandler(db)
	machineH := handlers.NewMachineHandler(db)
	storageH := handlers.NewStorageHandler(db, encKey)
	jobH     := handlers.NewBackupJobHandler(db, grpcSrv)
	runH     := handlers.NewBackupRunHandler(db, grpcSrv)
	restoreH := handlers.NewRestoreHandler(db, grpcSrv)
	alertH   := handlers.NewAlertHandler(db)
	dashH    := handlers.NewDashboardHandler(db)
	userH    := handlers.NewUserHandler(db)

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	// Public
	auth := api.Group("/auth")
	auth.POST("/register", authH.Register)
	auth.POST("/login", authH.Login)
	auth.POST("/refresh", authH.Refresh)
	auth.POST("/logout", authH.Logout)

	// Agent self-registration (agent token, not JWT)
	api.POST("/agents/register", agentH.Register)

	// Agent-authenticated endpoints (agent token, not JWT)
	// These sit outside the JWT authed group intentionally
	api.PUT("/backup-runs/:id/status", runH.UpdateStatus)
	api.POST("/backup-runs/:id/artifacts", runH.RegisterArtifact)
	api.POST("/artifacts/:artifact_id/chunks", runH.RegisterChunk)
	api.PUT("/restores/:id/status", restoreH.UpdateStatus)

	// Agent polling endpoints (agent token, not JWT) for agents that cannot
	// hold a persistent gRPC stream. Poll runs/restores, claim, fetch config.
	api.GET("/agent/runs", agentH.PendingRuns)
	api.POST("/agent/runs/:id/claim", agentH.ClaimRun)
	api.GET("/agent/jobs/:job_id/config", agentH.JobConfig)
	api.GET("/agent/restores", agentH.PendingRestores)
	api.POST("/agent/restores/:id/claim", agentH.ClaimRestore)

	// JWT-authenticated routes
	authed := api.Group("")
	authed.Use(middleware.Auth(cfg.JWTSecretKey))
	authed.Use(middleware.Audit(db))

	authed.GET("/me", userH.Me)
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

	// Backup runs
	org.GET("/backup-runs", runH.List)
	org.GET("/backup-runs/:id", runH.Get)
	org.POST("/backup-runs/:id/cancel", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), runH.Cancel)
	org.GET("/backup-runs/:id/artifacts", runH.ListArtifacts)

	// Restores
	org.GET("/restores", restoreH.List)
	org.POST("/restores", middleware.RequireRole(models.RoleOwner, models.RoleAdmin, models.RoleOperator), restoreH.Create)
	org.GET("/restores/:id", restoreH.Get)

	// Alerts
	org.GET("/alerts", alertH.List)
	org.PUT("/alerts/:id/read", alertH.MarkRead)

	// Dashboard
	org.GET("/dashboard", dashH.Overview)
	org.GET("/dashboard/health", dashH.BackupHealth)

	return r
}
