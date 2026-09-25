package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/algorithm"
	"pki-certificate-rollover-impact/backend/internal/config"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/database"
	"pki-certificate-rollover-impact/backend/internal/handler"
	"pki-certificate-rollover-impact/backend/internal/middleware"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/router"
	"pki-certificate-rollover-impact/backend/internal/service"
	"pki-certificate-rollover-impact/backend/internal/util"
)

func main() {
	logger := util.NewRedactingLogger(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := database.Close(db); closeErr != nil {
			logger.Error("database close failed", "error", closeErr)
		}
	}()
	transactions := repository.NewTransactionManager(db)
	anchors := repository.NewTrustAnchorRepository(db)
	chains := repository.NewCertificateChainRepository(db)
	servicesRepo := repository.NewDependentServiceRepository(db)
	scenarios := repository.NewRolloverScenarioRepository(db)
	audits := repository.NewAuditRepository(db)
	users := repository.NewUserRepository(db)
	anchorHandler := handler.NewTrustAnchorHandler(service.NewTrustAnchorService(anchors, audits, transactions))
	chainHandler := handler.NewCertificateChainHandler(service.NewCertificateChainService(chains, anchors, audits, transactions))
	dependentHandler := handler.NewDependentServiceHandler(service.NewDependentServiceService(servicesRepo, chains, anchors, audits, transactions))
	scenarioHandler := handler.NewRolloverScenarioHandler(service.NewRolloverScenarioService(scenarios, anchors, chains, servicesRepo, audits, transactions))
	auditHandler := handler.NewAuditHandler(service.NewAuditService(audits))
	auth := middleware.NewAuthenticator(users, cfg)
	loginLimiter := middleware.NewRateLimiter(cfg.LoginLimitPerMinute)
	certificateLimiter := middleware.NewRateLimiter(cfg.CertificateLimitPerMinute)
	simulationLimiter := middleware.NewRateLimiter(cfg.SimulationLimitPerMinute)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Recovery(logger), middleware.ErrorHandler(logger))
	engine.GET("/healthz", func(c *gin.Context) {
		sqlDB, dbErr := db.DB()
		if dbErr != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			util.Fail(c, util.NewError(http.StatusInternalServerError, util.CodeInternal, "database is unavailable"))
			return
		}
		util.Success(c, http.StatusOK, gin.H{"status": "healthy", "service": "pki-certificate-rollover-impact", "display_name": "CertRollover PKI 轮换推演台", "time": time.Now().UTC()})
	})
	v1 := engine.Group("/api/v1")
	v1.POST("/auth/login", loginLimiter.Middleware("login"), auth.Login)
	api := v1.Group("")
	api.Use(auth.RequireAuth(), middleware.Audit(logger))
	router.RegisterTrustAnchorRoutes(api, anchorHandler, certificateLimiter)
	router.RegisterCertificateChainRoutes(api, chainHandler, certificateLimiter)
	router.RegisterDependentServiceRoutes(api, dependentHandler)
	router.RegisterRolloverScenarioRoutes(api, scenarioHandler, simulationLimiter)
	api.GET("/audit-logs", middleware.RequirePermission(constants.PermissionAuditRead), auditHandler.List)
	api.GET("/meta/enums", middleware.RequirePermission(constants.PermissionRead), func(c *gin.Context) {
		util.Success(c, http.StatusOK, gin.H{"certificate_states": constants.CertificateStateValues(), "scenario_states": constants.ScenarioStateValues(), "chain_states": []constants.ChainState{constants.ChainImported, constants.ChainValidated, constants.ChainDeprecated, constants.ChainRevoked}, "service_states": []constants.ServiceState{constants.ServiceActive, constants.ServiceInactive}, "roles": constants.RoleValues(), "algorithm_version": algorithm.Version})
	})
	engine.NoRoute(func(c *gin.Context) {
		util.Fail(c, util.NewError(http.StatusNotFound, util.CodeNotFound, "route was not found"))
	})
	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	logger.Info("server started", "port", cfg.Port, "database_driver", cfg.DBDriver, "product", "CertRollover PKI 轮换推演台")
	select {
	case listenErr := <-serverErr:
		if !errors.Is(listenErr, http.ErrServerClosed) {
			return listenErr
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
	}
	return nil
}
