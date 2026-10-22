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

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"abet/auth-service/config"
	httpAdapter "abet/auth-service/internal/adapter/inbound/http"
	"abet/auth-service/internal/adapter/inbound/http/handler"
	"abet/auth-service/internal/adapter/outbound/google"
	"abet/auth-service/internal/adapter/outbound/jwt"
	"abet/auth-service/internal/adapter/outbound/mainclient"
	"abet/auth-service/internal/adapter/outbound/postgres"
	"abet/auth-service/internal/adapter/outbound/telemetry"
	"abet/auth-service/internal/application/commands"
	"abet/auth-service/internal/application/queries"
	"abet/auth-service/internal/core/domain"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	slog.SetDefault(slog.New(telemetry.NewTraceHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))))
	slog.Info("starting auth-service", "port", cfg.Port)

	// Tracing (no-op unless OTEL_ENABLED=true); must run before the router/clients are built.
	telCfg := telemetry.LoadConfig("auth-service")
	shutdownTelemetry, err := telemetry.Setup(context.Background(), telCfg)
	if err != nil {
		slog.Warn("tracing disabled: setup failed", "error", err)
	}
	defer func() { _ = shutdownTelemetry(context.Background()) }()
	if telCfg.Enabled {
		slog.Info("tracing enabled", "endpoint", telCfg.Endpoint, "sample_ratio", telCfg.SampleRatio)
	}

	policy, err := domain.NewEmailDomainPolicy(cfg.AllowedDomains)
	if err != nil {
		return err
	}

	// 0. Database: migrate, then connect
	if err := postgres.Migrate(cfg.DB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	ctx := context.Background()
	pool, err := postgres.Connect(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	// 1. Initialize Driven (Outbound) Adapters
	sessionRepo := postgres.NewSessionRepository(pool)
	tokenService := jwt.NewTokenService(cfg.SessionSecret)
	googleProvider := google.NewProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL).
		WithHTTPClient(&http.Client{Timeout: 15 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)})
	mainClient := mainclient.NewHTTPClient(cfg.MainServiceURL, cfg.InternalAPIToken, cfg.MainServiceTimeout)
	if !googleProvider.Configured() {
		slog.Warn("GOOGLE_CLIENT_ID is empty: /api/v1/auth/google/login will return 503")
	}

	// 2. Initialize Application Layer Use Cases (CQRS)
	startLoginUseCase := commands.NewStartGoogleLoginHandler(googleProvider, policy)
	callbackUseCase := commands.NewGoogleCallbackHandler(googleProvider, mainClient, sessionRepo, tokenService, policy, cfg.SessionTTL, nil)
	logoutUseCase := commands.NewLogoutHandler(tokenService, sessionRepo, nil)
	cleanupUseCase := commands.NewCleanupSessionsHandler(sessionRepo, nil)
	verifyUseCase := queries.NewVerifySessionHandler(tokenService, sessionRepo, nil)

	// 3. Initialize Driving (Inbound) HTTP Adapters
	healthHandler := handler.NewHealthHandler(sessionRepo)
	authHandler := handler.NewAuthHandler(startLoginUseCase, callbackUseCase, logoutUseCase, verifyUseCase, handler.AuthConfig{
		PublicBaseURL: cfg.PublicBaseURL,
		CookieSecure:  cfg.CookieSecure,
		SessionSecret: cfg.SessionSecret,
	}, nil)

	// 4. Setup Router
	router, err := httpAdapter.SetupRouter(
		httpAdapter.RouterConfig{
			EnablePprof:   cfg.EnablePprof,
			EnableSwagger: cfg.EnableSwagger,
		},
		healthHandler,
		authHandler,
	)
	if err != nil {
		return fmt.Errorf("setup router: %w", err)
	}

	// 5. Background cleanup of long-expired sessions (on startup and hourly)
	cleanupCtx, stopCleanup := context.WithCancel(ctx)
	defer stopCleanup()
	go runSessionCleanup(cleanupCtx, cleanupUseCase)

	// 6. Start Server with Graceful Shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	slog.Info("server listening", "addr", srv.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
	}
	slog.Info("shutting down server gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}
	slog.Info("server exited")
	return nil
}

func runSessionCleanup(ctx context.Context, uc *commands.CleanupSessionsHandler) {
	cleanup := func() {
		deleted, err := uc.Execute(ctx)
		if err != nil {
			slog.Error("session cleanup failed", "error", err)
			return
		}
		slog.Info("session cleanup finished", "deleted", deleted)
	}

	cleanup()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}
