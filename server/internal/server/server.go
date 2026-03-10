package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/config"
	"github.com/rendis/feature-evaluator/internal/domain/apikey"
	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/evaluation"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	evalmetrics "github.com/rendis/feature-evaluator/internal/domain/metrics"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/domain/schedule"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/domain/tag"
	"github.com/rendis/feature-evaluator/internal/domain/tier"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/internal/engine"
	"github.com/rendis/feature-evaluator/internal/external"
	"github.com/rendis/feature-evaluator/internal/handler"
	"github.com/rendis/feature-evaluator/internal/incomingauth"
	"github.com/rendis/feature-evaluator/internal/secrets"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/internal/storage/postgres"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

// Server holds all dependencies and the HTTP server.
type Server struct {
	cfg              *config.Config
	httpServer       *http.Server
	postgres         *postgres.Client
	redis            *redisclient.Client
	metricsCollector *evalmetrics.Collector
	scheduleWorker   *schedule.Worker
}

// New creates a new Server with all routes configured.
//
//nolint:funlen // Wiring all repositories, services, handlers, and routes is inherently a large composition root.
func New(cfg *config.Config, postgresDB *postgres.Client, redis *redisclient.Client) *Server {
	if cfg.Log.Format == "json" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.BodySizeLimit(1<<20), // 1MB global default
		middleware.Logging(),
		middleware.CORS(cfg.CORS.AllowOrigins),
		middleware.TenantExtractor(),
		middleware.WorkspaceResolver(),
	)

	// Repositories
	memberRepo := postgres.NewMemberRepo(postgresDB)
	featureRepo := postgres.NewFeatureRepo(postgresDB)
	segmentRepo := postgres.NewSegmentRepo(postgresDB)
	segmentRecordRepo := postgres.NewSegmentRecordRepo(postgresDB)
	evalErrorRepo := postgres.NewEvalErrorRepo(postgresDB)
	apiKeyRepo := postgres.NewAPIKeyRepo(postgresDB)
	authProfileRepo := postgres.NewAuthProfileRepo(postgresDB)
	externalAPIRepo := postgres.NewExternalAPIRepo(postgresDB)
	tagRepo := postgres.NewTagRepo(postgresDB)
	tierRepo := postgres.NewTierRepo(postgresDB)
	tierIconRepo := postgres.NewTierIconRepo(postgresDB)
	packRepo := postgres.NewPackRepo(postgresDB)
	packActivationRepo := postgres.NewPackActivationRepo(postgresDB)
	changelogRepo := postgres.NewChangelogRepo(postgresDB)
	workspaceRepo := postgres.NewWorkspaceRepo(postgresDB)
	scheduleRepo := postgres.NewScheduleRepo(postgresDB)
	experimentRepo := postgres.NewExperimentRepo(postgresDB)
	exposureRepo := postgres.NewExposureRepo(postgresDB)
	conversionRepo := postgres.NewConversionRepo(postgresDB)

	// Engine
	exprEngine, err := engine.New()
	if err != nil {
		slog.Error("creating expression engine", "error", err)
		panic(fmt.Sprintf("creating expression engine: %v", err))
	}

	// Services
	secretCipher, err := secrets.NewCipher(cfg.Auth.SecretsMasterKey)
	if err != nil {
		slog.Error("creating secret cipher", "error", err)
		panic(fmt.Sprintf("creating secret cipher: %v", err))
	}
	memberSvc := member.NewService(memberRepo)
	featureSvc := feature.NewService(featureRepo)
	segmentCache := redisclient.NewSegmentCache(redis)
	segmentSvc := segment.NewService(segmentRepo, segmentRecordRepo, segmentCache)
	segmentSvc.SetTxManager(postgresDB)
	auditSvc := audit.NewService(evalErrorRepo)
	apiKeySvc := apikey.NewService(apiKeyRepo)
	authProfileSvc := authprofile.NewService(authProfileRepo, secretCipher)
	externalAPISvc := externalapi.NewService(externalAPIRepo, secretCipher)
	tagSvc := tag.NewService(tagRepo)
	tierSvc := tier.NewService(tierRepo, tierIconRepo)
	packCache := redisclient.NewPackCache(redis)
	packSvc := pack.NewService(packRepo, packActivationRepo, featureRepo, packCache)
	changelogSvc := changelog.NewService(changelogRepo)
	workspaceSvc := workspace.NewService(workspaceRepo)
	scheduleSvc := schedule.NewService(scheduleRepo)
	experimentCache := redisclient.NewExperimentCache(redis)
	experimentSvc := experiment.NewService(experimentRepo, exposureRepo, conversionRepo, featureSvc, experimentCache)
	experimentSvc.SetTxManager(postgresDB)
	evalSvc := evaluation.NewService(featureRepo, segmentSvc, auditSvc, exprEngine)
	evalSvc.SetPackService(packSvc)
	evalSvc.SetExperimentService(experimentSvc)
	authValidator := incomingauth.NewValidator(redis, authProfileSvc)
	evalSvc.SetAuthValidator(authValidator)

	// External caller (created early so resolver can reference it)
	extCaller := external.NewCaller(redis, secretCipher)
	extApiResolver := external.NewExternalApiResolver(externalAPISvc, extCaller)
	evalSvc.SetExternalApiResolver(extApiResolver)

	// Metrics collector
	metricsCollector := evalmetrics.NewCollector(redis)
	metricsCollector.Start()

	// Schedule worker
	scheduleWorker := schedule.NewWorker(scheduleRepo, featureSvc, changelogSvc)
	scheduleWorker.Start()

	// Hook: cache access metrics
	redis.OnCacheAccess = func(hit bool) {
		metricsCollector.RecordCacheAccess(hit)
	}

	// Hook: segment lookup metrics
	evalSvc.SetOnSegmentLookup(func(segmentKey string) {
		metricsCollector.RecordSegmentLookup(segmentKey)
	})

	// External caller metrics hooks
	extCaller.SetOnLatency(func(d time.Duration) {
		metricsCollector.RecordExternalLatency(d)
	})
	extCaller.CBManager().SetOnStateChange(func(opened bool) {
		metricsCollector.RecordCircuitBreakerEvent(opened)
	})
	// JWT validator
	jwtValidator := middleware.NewJWTValidator(cfg.OIDC.Issuer, cfg.OIDC.Audience)

	// Rate limiter
	rateLimiter := middleware.NewRateLimiter(redis.Underlying())
	rateLimiter.SetOnReject(func() {
		metricsCollector.RecordRateLimitReject()
	})

	// Handlers
	healthHandler := handler.NewHealthHandler(postgresDB, redis)
	memberHandler := handler.NewMemberHandler(memberSvc)
	authProfileHandler := handler.NewAuthProfileHandler(authProfileSvc, authValidator, extCaller)
	externalAPIHandler := handler.NewExternalAPIHandler(externalAPISvc, extCaller)
	featureHandler := handler.NewFeatureHandler(featureSvc, tagSvc, packSvc, tierSvc, changelogSvc)
	tagHandler := handler.NewTagHandler(tagSvc)
	tierHandler := handler.NewTierHandler(tierSvc)
	packHandler := handler.NewPackHandler(packSvc, tierSvc, changelogSvc)
	ruleHandler := handler.NewRuleHandler(featureSvc, segmentSvc, externalAPISvc, extApiResolver, exprEngine, changelogSvc)
	changelogHandler := handler.NewChangelogHandler(changelogSvc)
	evalHandler := handler.NewEvalHandler(evalSvc, metricsCollector)
	ofrepHandler := handler.NewOFREPHandler(evalSvc, metricsCollector)
	segmentHandler := handler.NewSegmentHandler(segmentSvc, changelogSvc)
	auditHandler := handler.NewAuditHandler(auditSvc)
	dashboardHandler := handler.NewDashboardHandler(featureSvc, segmentSvc, auditSvc, metricsCollector, postgresDB, redis)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)
	metricsHandler := handler.NewMetricsHandler(metricsCollector)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceSvc, memberSvc)
	scheduleHandler := handler.NewScheduleHandler(scheduleSvc)
	experimentHandler := handler.NewExperimentHandler(experimentSvc, changelogSvc)

	// Routes under /features base path
	base := router.Group("/features")

	// Health endpoints (no auth)
	base.GET("/healthz", healthHandler.Liveness)
	base.GET("/readyz", healthHandler.Readiness)

	// Eval routes (require Bearer or API key) + rate limit
	evalGroup := base.Group("")
	evalGroup.Use(middleware.RequireActiveWorkspace(workspaceSvc))
	evalGroup.Use(middleware.EvalAuth(cfg.Auth, jwtValidator, apiKeySvc))
	evalGroup.Use(rateLimiter.Limit("fe:rl:eval:", cfg.RateLimit.EvalPerSecond, func(c *gin.Context) string {
		if apiKey := c.GetHeader("X-Api-Key"); apiKey != "" {
			return apiKey
		}
		if userEmail := middleware.GetUserEmail(c); userEmail != "" {
			return userEmail
		}
		return c.ClientIP()
	}))
	evalGroup.POST("/eval", evalHandler.Evaluate)
	evalGroup.POST("/eval/bulk", evalHandler.BulkEvaluate)
	evalGroup.POST("/eval/active", evalHandler.EvaluateAll)
	evalGroup.POST("/eval/conversions", experimentHandler.RecordConversion)

	// OFREP routes (OpenFeature Remote Evaluation Protocol)
	ofrep := router.Group("/ofrep/v1")
	ofrep.Use(
		middleware.RequireActiveWorkspace(workspaceSvc),
		middleware.EvalAuth(cfg.Auth, jwtValidator, apiKeySvc),
		rateLimiter.Limit("fe:rl:eval:", cfg.RateLimit.EvalPerSecond, func(c *gin.Context) string {
			if apiKey := c.GetHeader("X-Api-Key"); apiKey != "" {
				return apiKey
			}
			if userEmail := middleware.GetUserEmail(c); userEmail != "" {
				return userEmail
			}
			return c.ClientIP()
		}),
	)
	ofrep.POST("/evaluate/flags/:key", ofrepHandler.EvaluateSingle)
	ofrep.POST("/evaluate/flags", ofrepHandler.EvaluateBulk)

	// Admin routes (require auth via JWT or admin API key) + rate limit
	admin := base.Group("/admin")
	admin.Use(middleware.RequireActiveWorkspace(workspaceSvc))
	admin.Use(middleware.AdminAuth(cfg.Auth, memberSvc, jwtValidator, apiKeySvc))
	admin.Use(rateLimiter.Limit("fe:rl:admin:", cfg.RateLimit.AdminPerSecond, func(c *gin.Context) string {
		return middleware.GetUserEmail(c)
	}))

	// Members
	admin.GET("/members/me", memberHandler.GetMe)

	membersRead := admin.Group("/members")
	membersRead.Use(middleware.RequirePermission(member.PermMembersRead))
	membersRead.GET("", memberHandler.List)

	membersManage := admin.Group("/members")
	membersManage.Use(middleware.RequirePermission(member.PermMembersManage))
	membersManage.POST("", memberHandler.Create)
	membersManage.PUT("/:id/role", memberHandler.UpdateRole)
	membersManage.DELETE("/:id", memberHandler.Delete)

	ownerOnly := admin.Group("/members")
	ownerOnly.Use(middleware.RequirePermission(member.PermOwnershipTransfer))
	ownerOnly.POST("/:id/transfer-ownership", memberHandler.TransferOwnership)

	// API Keys (admin-level)
	apiKeysGroup := admin.Group("/api-keys")
	apiKeysGroup.Use(middleware.RequirePermission(member.PermMembersManage))
	apiKeysGroup.POST("", apiKeyHandler.Create)
	apiKeysGroup.GET("", apiKeyHandler.List)
	apiKeysGroup.PUT("/:id/rotate", apiKeyHandler.Rotate)
	apiKeysGroup.DELETE("/:id", apiKeyHandler.Revoke)

	// Environments (static list for frontend dropdowns)
	admin.GET("/environments", featureHandler.ListEnvironments)

	// Tags
	tagsRead := admin.Group("/tags")
	tagsRead.Use(middleware.RequirePermission(member.PermFeaturesRead))
	tagsRead.GET("", tagHandler.List)

	authProfilesRead := admin.Group("/auth-profiles")
	authProfilesRead.Use(middleware.RequirePermission(member.PermFeaturesRead))
	authProfilesRead.GET("", authProfileHandler.List)
	authProfilesRead.GET("/:key", authProfileHandler.Get)

	externalAPIsRead := admin.Group("/external-apis")
	externalAPIsRead.Use(middleware.RequirePermission(member.PermFeaturesRead))
	externalAPIsRead.GET("", externalAPIHandler.List)
	externalAPIsRead.GET("/expression-profile", externalAPIHandler.ExpressionProfile)
	externalAPIsRead.GET("/:key", externalAPIHandler.Get)
	externalAPIsRead.POST("/expression/validate", externalAPIHandler.ValidateExpression)

	tagsWrite := admin.Group("/tags")
	tagsWrite.Use(middleware.RequirePermission(member.PermFeaturesWrite))
	tagsWrite.POST("", tagHandler.Create)
	tagsWrite.PUT("/:key", tagHandler.Update)
	tagsWrite.DELETE("/:key", tagHandler.Delete)

	// Tiers
	tiersRead := admin.Group("/tiers")
	tiersRead.Use(middleware.RequirePermission(member.PermFeaturesRead))
	tiersRead.GET("", tierHandler.List)
	tiersRead.GET("/:key", tierHandler.Get)
	tiersRead.GET("/icons", tierHandler.ListIcons)

	tiersWrite := admin.Group("/tiers")
	tiersWrite.Use(middleware.RequirePermission(member.PermFeaturesWrite))
	tiersWrite.POST("", tierHandler.Create)
	tiersWrite.PUT("/:key", tierHandler.Update)
	tiersWrite.DELETE("/:key", tierHandler.Delete)
	tiersWrite.POST("/icons", tierHandler.UploadIcon)
	tiersWrite.DELETE("/icons/:id", tierHandler.DeleteIcon)

	authProfilesWrite := admin.Group("/auth-profiles")
	authProfilesWrite.Use(middleware.RequirePermission(member.PermSettingsManage))
	authProfilesWrite.POST("", authProfileHandler.Create)
	authProfilesWrite.POST("/test", authProfileHandler.Test)
	authProfilesWrite.PUT("/:key", authProfileHandler.Update)
	authProfilesWrite.DELETE("/:key", authProfileHandler.Delete)

	externalAPIsWrite := admin.Group("/external-apis")
	externalAPIsWrite.Use(middleware.RequirePermission(member.PermSettingsManage))
	externalAPIsWrite.POST("", externalAPIHandler.Create)
	externalAPIsWrite.POST("/test", externalAPIHandler.Test)
	externalAPIsWrite.PUT("/:key", externalAPIHandler.Update)
	externalAPIsWrite.DELETE("/:key", externalAPIHandler.Delete)

	// Packs
	packsRead := admin.Group("/packs")
	packsRead.Use(middleware.RequirePermission(member.PermPacksRead))
	packsRead.GET("", packHandler.List)
	packsRead.GET("/:key", packHandler.Get)
	packsRead.GET("/:key/activations", packHandler.ListActivations)
	packsRead.GET("/by-target", packHandler.ByTarget)

	packsWrite := admin.Group("/packs")
	packsWrite.Use(middleware.RequirePermission(member.PermPacksWrite))
	packsWrite.POST("", packHandler.Create)
	packsWrite.PUT("/:key", packHandler.Update)
	packsWrite.DELETE("/:key", packHandler.Delete)
	packsWrite.PATCH("/:key/toggle", packHandler.Toggle)
	packsWrite.POST("/:key/activate", packHandler.Activate)
	packsWrite.DELETE("/:key/activate", packHandler.Deactivate)

	// Features
	featuresRead := admin.Group("/features")
	featuresRead.Use(middleware.RequirePermission(member.PermFeaturesRead))
	featuresRead.GET("", featureHandler.List)
	featuresRead.GET("/:key", featureHandler.Get)
	featuresRead.GET("/:key/rules", ruleHandler.List)
	featuresRead.GET("/:key/expression-schema", ruleHandler.FeatureExpressionSchema)

	featuresWrite := admin.Group("/features")
	featuresWrite.Use(middleware.RequirePermission(member.PermFeaturesWrite))
	featuresWrite.POST("", featureHandler.Create)
	featuresWrite.PUT("/:key", featureHandler.Update)
	featuresWrite.DELETE("/:key", featureHandler.Delete)
	featuresWrite.PATCH("/:key/toggle", featureHandler.Toggle)
	featuresWrite.POST("/:key/rules", ruleHandler.Create)
	featuresWrite.PUT("/:key/rules/:ruleId", ruleHandler.Update)
	featuresWrite.DELETE("/:key/rules/:ruleId", ruleHandler.Delete)
	featuresWrite.PUT("/:key/rules/reorder", ruleHandler.Reorder)
	featuresWrite.POST("/:key/expression/test", ruleHandler.FeatureTestExpression)
	featuresWrite.POST("/:key/schedules", scheduleHandler.Create)

	featuresRead.GET("/:key/schedules", scheduleHandler.List)

	// Schedule cancellation (by schedule ID, not feature key)
	schedulesWrite := admin.Group("/schedules")
	schedulesWrite.Use(middleware.RequirePermission(member.PermFeaturesWrite))
	schedulesWrite.DELETE("/:id", scheduleHandler.Cancel)

	// Expression tools
	exprGroup := admin.Group("/expression")
	exprGroup.Use(middleware.RequirePermission(member.PermFeaturesRead))
	exprGroup.POST("/validate", ruleHandler.ValidateExpression)
	exprGroup.POST("/test", ruleHandler.TestExpression)
	exprGroup.GET("/schema", ruleHandler.ExpressionSchema)

	// Segments
	segmentsRead := admin.Group("/segments")
	segmentsRead.Use(middleware.RequirePermission(member.PermSegmentsRead))
	segmentsRead.GET("", segmentHandler.List)
	segmentsRead.GET("/:key", segmentHandler.Get)
	segmentsRead.GET("/:key/schema", segmentHandler.GetSchema)
	segmentsRead.GET("/:key/records", segmentHandler.ListRecords)

	segmentsWrite := admin.Group("/segments")
	segmentsWrite.Use(middleware.RequirePermission(member.PermSegmentsWrite))
	segmentsWrite.POST("", segmentHandler.Create)
	segmentsWrite.PUT("/:key", segmentHandler.Update)
	segmentsWrite.DELETE("/:key", segmentHandler.Delete)
	segmentsWrite.POST("/:key/data/import", segmentHandler.ImportData)

	// Dashboard
	dashboardGroup := admin.Group("/dashboard")
	dashboardGroup.Use(middleware.RequirePermission(member.PermFeaturesRead))
	dashboardGroup.GET("/stats", dashboardHandler.Stats)
	dashboardGroup.GET("/activity", dashboardHandler.Activity)
	dashboardGroup.GET("/error-summary", dashboardHandler.ErrorSummary)
	dashboardGroup.GET("/operations", dashboardHandler.Operations)

	// Experiments
	experimentsRead := admin.Group("/experiments")
	experimentsRead.Use(middleware.RequirePermission(member.PermExperimentsRead))
	experimentsRead.GET("", experimentHandler.List)
	experimentsRead.GET("/:id", experimentHandler.Get)
	experimentsRead.GET("/:id/results", experimentHandler.GetResults)

	experimentsWrite := admin.Group("/experiments")
	experimentsWrite.Use(middleware.RequirePermission(member.PermExperimentsWrite))
	experimentsWrite.POST("", experimentHandler.Create)
	experimentsWrite.PUT("/:id", experimentHandler.Update)
	experimentsWrite.POST("/:id/start", experimentHandler.Start)
	experimentsWrite.POST("/:id/pause", experimentHandler.Pause)
	experimentsWrite.POST("/:id/complete", experimentHandler.Complete)
	experimentsWrite.POST("/:id/declare-winner", experimentHandler.DeclareWinner)

	// Dashboard Metrics
	metricsGroup := admin.Group("/dashboard/metrics")
	metricsGroup.Use(middleware.RequirePermission(member.PermAuditRead))
	metricsGroup.GET("/overview", metricsHandler.Overview)
	metricsGroup.GET("/features", metricsHandler.Features)
	metricsGroup.GET("/reasons", metricsHandler.Reasons)
	metricsGroup.GET("/tenants", metricsHandler.Tenants)
	metricsGroup.GET("/environments", metricsHandler.Environments)
	metricsGroup.GET("/cache", metricsHandler.Cache)
	metricsGroup.GET("/external", metricsHandler.External)

	// Audit
	auditGroup := admin.Group("/audit")
	auditGroup.Use(middleware.RequirePermission(member.PermAuditRead))
	auditGroup.GET("/errors", auditHandler.ListErrors)

	// Changelog (read-only, audit permission)
	changelogGroup := admin.Group("/changelog")
	changelogGroup.Use(middleware.RequirePermission(member.PermAuditRead))
	changelogGroup.GET("", changelogHandler.List)
	changelogGroup.GET("/:entityType/:entityKey", changelogHandler.ListByEntity)

	// Workspaces
	workspaceReadGroup := base.Group("/admin/workspaces")
	workspaceReadGroup.Use(middleware.WorkspaceReadAuth(cfg.Auth, jwtValidator))
	workspaceReadGroup.GET("", workspaceHandler.List)
	workspaceReadGroup.GET("/:key", workspaceHandler.Get)

	workspaceWriteGroup := base.Group("/admin/workspaces")
	workspaceWriteGroup.Use(middleware.WorkspaceManageAuth(cfg.Auth, memberSvc, workspaceSvc, jwtValidator))
	workspaceWriteGroup.Use(middleware.RequirePermission(member.PermOwnershipTransfer))
	workspaceWriteGroup.POST("", workspaceHandler.Create)
	workspaceWriteGroup.PUT("/:key", workspaceHandler.Update)
	workspaceWriteGroup.POST("/:key/archive", workspaceHandler.Archive)
	workspaceWriteGroup.POST("/:key/restore", workspaceHandler.Restore)
	workspaceWriteGroup.DELETE("/:key", workspaceHandler.Delete)

	srv := &Server{
		cfg:              cfg,
		postgres:         postgresDB,
		redis:            redis,
		metricsCollector: metricsCollector,
		scheduleWorker:   scheduleWorker,
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}

	return srv
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	slog.Info("starting HTTP server", "port", s.cfg.Server.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down HTTP server")
	if s.scheduleWorker != nil {
		s.scheduleWorker.Stop()
	}
	if s.metricsCollector != nil {
		slog.Info("stopping metrics collector")
		s.metricsCollector.Stop()
	}
	return s.httpServer.Shutdown(ctx)
}
