package router

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dipesh/bifrost/backend/config"
	"github.com/dipesh/bifrost/backend/internal/admin"
	"github.com/dipesh/bifrost/backend/internal/agent"
	"github.com/dipesh/bifrost/backend/internal/auth"
	"github.com/dipesh/bifrost/backend/internal/install"
	"github.com/dipesh/bifrost/backend/internal/monitoring"
	"github.com/dipesh/bifrost/backend/internal/onboarding"
	shareddb "github.com/dipesh/bifrost/backend/internal/shared/database"
	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
	sharedmiddleware "github.com/dipesh/bifrost/backend/internal/shared/middleware"
)

type DataStore interface {
	auth.Store
	admin.Store
	onboarding.Store
	monitoring.Store
	agent.Store
}

func New(cfg config.Config, dataStore DataStore) *gin.Engine {
	inspector, err := sharedhttp.NewRequestInspector(cfg.TrustedProxies)
	if err != nil {
		log.Fatalf("configure trusted proxies: %v", err)
	}

	timeouts := shareddb.QueryTimeouts{
		Read:   cfg.DBReadTimeout,
		Write:  cfg.DBWriteTimeout,
		Ingest: cfg.DBIngestTimeout,
	}

	engine := gin.New()
	if len(inspector.TrustedProxyStrings()) == 0 {
		if err := engine.SetTrustedProxies(nil); err != nil {
			log.Fatalf("disable trusted proxies: %v", err)
		}
	} else if err := engine.SetTrustedProxies(inspector.TrustedProxyStrings()); err != nil {
		log.Fatalf("set trusted proxies: %v", err)
	}
	engine.Use(
		sharedmiddleware.Logger(),
		sharedmiddleware.Recovery(),
		sharedmiddleware.RequestID(),
		sharedmiddleware.SecurityLog(),
		sharedmiddleware.BodyLimit(),
		sharedmiddleware.InFlightLimit(),
	)

	authService := auth.NewService(auth.NewRepository(dataStore, timeouts))
	authHandler := auth.NewHandler(authService, inspector)

	adminService := admin.NewService(admin.NewRepository(dataStore, timeouts))
	adminHandler := admin.NewHandler(adminService)

	onboardingService := onboarding.NewService(onboarding.NewRepository(dataStore, timeouts), cfg.AgentBackendURL, cfg.AgentDockerImage)
	onboardingHandler := onboarding.NewHandler(onboardingService)

	monitoringService := monitoring.NewService(monitoring.NewRepository(dataStore, timeouts))
	monitoringHandler := monitoring.NewHandler(monitoringService)

	agentService := agent.NewService(agent.NewRepository(dataStore, timeouts))
	agentHandler := agent.NewHandler(agentService)

	installService := install.NewService(install.Config{
		AgentDockerImage: cfg.AgentDockerImage,
		AgentBinaryPath:  cfg.AgentBinaryPath,
		AgentSourceDir:   cfg.AgentSourceDir,
	}, inspector)
	installHandler := install.NewHandler(installService)

	engine.GET("/health", health)

	api := engine.Group("/api/v1")
	api.Use(sharedmiddleware.NewPublicRateLimit())
	{
		install.RegisterRoutes(api, installHandler)
		auth.RegisterPublicRoutes(api, authHandler)
		agent.RegisterRoutes(api, agentHandler)
	}

	protected := api.Group("/")
	protected.Use(auth.AuthRequired(authService), sharedmiddleware.NewAuthenticatedRateLimit())
	{
		auth.RegisterProtectedRoutes(protected, authHandler)
		monitoring.RegisterRoutes(protected, monitoringHandler)
	}

	adminGroup := protected.Group("/admin")
	adminGroup.Use(auth.AdminRequired())
	{
		admin.RegisterRoutes(adminGroup, adminHandler)
		onboarding.RegisterRoutes(adminGroup, onboardingHandler)
	}

	return engine
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, sharedhttp.Success(gin.H{
		"service": "bifrost-backend",
		"status":  "ok",
	}))
}
