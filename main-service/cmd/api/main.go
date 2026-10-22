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

	"abet/main-service/config"
	httpAdapter "abet/main-service/internal/adapter/inbound/http"
	"abet/main-service/internal/adapter/inbound/http/handler"
	"abet/main-service/internal/adapter/inbound/scheduler"
	"abet/main-service/internal/adapter/outbound/postgres"
	"abet/main-service/internal/adapter/outbound/rabbitmq"
	"abet/main-service/internal/adapter/outbound/storage"
	"abet/main-service/internal/adapter/outbound/telemetry"
	"abet/main-service/internal/adapter/outbound/templates"
	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/application/queries"
	"abet/main-service/internal/application/queuehealth"
)

func main() {
	cfg := config.Load()
	setupLogger(cfg.LogLevel)
	// Tracing (no-op unless OTEL_ENABLED=true); must run before the router and the clients are built.
	telCfg := telemetry.LoadConfig("main-service")
	shutdownTelemetry, err := telemetry.Setup(context.Background(), telCfg)
	if err != nil {
		slog.Warn("tracing disabled: setup failed", "error", err)
	}
	if telCfg.Enabled {
		slog.Info("tracing enabled", "endpoint", telCfg.Endpoint, "sample_ratio", telCfg.SampleRatio)
	}
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	runErr := run(cfg)
	_ = shutdownTelemetry(context.Background()) // flush pending spans
	if runErr != nil {
		slog.Error("fatal", "error", runErr)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	ctx := context.Background()
	slog.Info("starting main-service", "port", cfg.Port)

	// Dev-only "act as user" (D28): needs DEV_ACT_AS_ENABLED=true AND APP_ENV=development.
	actAsEnabled := cfg.DevActAsEnabled()
	switch {
	case actAsEnabled:
		slog.Warn("dev act-as enabled")
	case cfg.DevActAsRequested:
		slog.Error("DEV_ACT_AS_ENABLED=true ignored: APP_ENV is not development, act-as stays OFF", "app_env", cfg.AppEnv)
	}

	// 1. Database: migrate, then connect.
	dbCfg := postgres.DBConfig{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser, Password: cfg.DBPassword, Name: cfg.DBName, SSLMode: cfg.DBSSLMode,
	}
	if err := postgres.Migrate(dbCfg); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	store, err := postgres.NewStore(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer store.Close()

	// 2. Outbound adapters.
	departmentRepo := postgres.NewDepartmentRepository(store)
	curriculumRepo := postgres.NewCurriculumRepository(store)
	userRepo := postgres.NewUserRepository(store)
	offeringRepo := postgres.NewOfferingRepository(store)
	accessRepo := postgres.NewAccessRepository(store)
	importRepo := postgres.NewImportRepository(store)
	coverRepo := postgres.NewCoverRepository(store)
	fileStore, err := storage.NewMinIO(storage.Config{
		Endpoint: cfg.MinIOEndpoint, AccessKey: cfg.MinIOAccessKey, SecretKey: cfg.MinIOSecretKey,
		Bucket: cfg.MinIOBucket, UseSSL: cfg.MinIOUseSSL,
	})
	if err != nil {
		return err
	}
	if err := ensureBucket(ctx, fileStore); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.UploadTmpDir, 0o700); err != nil {
		return fmt.Errorf("create upload temp dir: %w", err)
	}
	submissionRepo := postgres.NewSubmissionRepository(store)
	rabbitCfg := rabbitmq.Config{
		Host: cfg.RabbitMQHost, Port: cfg.RabbitMQPort, User: cfg.RabbitMQUser, Pass: cfg.RabbitMQPass,
		VHost: cfg.RabbitMQVHost, JobQueue: cfg.RabbitMQQueue, StatusQueue: cfg.RabbitMQStatusQueue,
	}
	jobPublisher := rabbitmq.NewPublisher(rabbitCfg)
	defer jobPublisher.Close()
	templateService := templates.New(cfg.InstructorPackageDir)

	// 3. Bootstrap super admins from SUPER_ADMIN_EMAILS.
	n, err := commands.NewBootstrapSuperAdminsHandler(userRepo).Execute(ctx, cfg.SuperAdminEmails)
	if err != nil {
		return fmt.Errorf("bootstrap super admins: %w", err)
	}
	slog.Info("super admins bootstrapped", "count", n)

	// 4. Application layer (CQRS use cases).
	createDepartment := commands.NewCreateDepartmentHandler(departmentRepo)
	updateDepartment := commands.NewUpdateDepartmentHandler(departmentRepo)
	deleteDepartment := commands.NewDeleteDepartmentHandler(departmentRepo)
	listDepartments := queries.NewListDepartmentsHandler(departmentRepo)

	createCurriculum := commands.NewCreateCurriculumHandler(curriculumRepo, departmentRepo)
	updateCurriculum := commands.NewUpdateCurriculumHandler(curriculumRepo, departmentRepo)
	deleteCurriculum := commands.NewDeleteCurriculumHandler(curriculumRepo, fileStore)
	listCurricula := queries.NewListCurriculaHandler(curriculumRepo)

	createUser := commands.NewCreateUserHandler(userRepo, departmentRepo, curriculumRepo, cfg.AllowedEmailDomains)
	updateUser := commands.NewUpdateUserHandler(userRepo, departmentRepo, curriculumRepo)
	deleteUser := commands.NewDeleteUserHandler(userRepo)
	listUsers := queries.NewListUsersHandler(userRepo)
	getMe := queries.NewGetMeHandler(userRepo)
	getActAsState := queries.NewGetActAsStateHandler(userRepo, actAsEnabled)
	loginSync := commands.NewLoginSyncHandler(userRepo)

	listOfferings := queries.NewListOfferingsHandler(offeringRepo)
	listOfferingYears := queries.NewListOfferingYearsHandler(offeringRepo)
	createOffering := commands.NewCreateOfferingHandler(offeringRepo, curriculumRepo)
	updateOffering := commands.NewUpdateOfferingHandler(offeringRepo, curriculumRepo)
	deleteOfferings := commands.NewDeleteOfferingsHandler(offeringRepo, fileStore)
	listAccess := queries.NewListAccessHandler(offeringRepo, accessRepo)
	setAccess := commands.NewSetAccessHandler(offeringRepo, accessRepo, userRepo)
	removeAccess := commands.NewRemoveAccessHandler(offeringRepo, accessRepo)
	maxImportBytes := int64(cfg.MaxImportMB) << 20
	previewImport := commands.NewPreviewImportHandler(importRepo, curriculumRepo, cfg.AllowedEmailDomains, maxImportBytes)
	applyImport := commands.NewApplyImportHandler(importRepo, curriculumRepo, cfg.AllowedEmailDomains, maxImportBytes)
	maxCoverBytes := int64(cfg.MaxCoverMB) << 20
	putCover := commands.NewPutCoverHandler(curriculumRepo, coverRepo, fileStore, maxCoverBytes)
	getCover := commands.NewGetCoverHandler(curriculumRepo, fileStore)
	deleteCover := commands.NewDeleteCoverHandler(curriculumRepo, coverRepo, fileStore)

	uploadSubmission := commands.NewUploadSubmissionHandler(submissionRepo, fileStore, jobPublisher, cfg.UploadTmpDir)
	processBuildStatus := commands.NewProcessBuildStatusHandler(submissionRepo, fileStore)
	getSubmission := queries.NewGetSubmissionHandler(submissionRepo)
	listSubmissions := queries.NewListSubmissionsHandler(submissionRepo)
	getPortfolioPDF := queries.NewGetPortfolioPDFHandler(submissionRepo, fileStore)
	listPortfolios := queries.NewListPortfoliosHandler(submissionRepo)
	downloadPortfolios := queries.NewDownloadPortfoliosHandler(submissionRepo, fileStore)

	// Background jobs: build-timeout reconciler, queue health, storage janitor.
	queueMonitor := queuehealth.New(5 * time.Minute)
	reconcileBuilds := commands.NewReconcileBuildsHandler(submissionRepo, fileStore, time.Duration(cfg.BuildTimeoutMinutes)*time.Minute)
	janitor := commands.NewJanitorHandler(submissionRepo, fileStore, time.Hour)
	jobs := scheduler.New(time.Duration(cfg.ReconcileIntervalSeconds)*time.Second, reconcileBuilds, janitor,
		rabbitmq.NewQueueInspector(rabbitCfg), queueMonitor)

	// 5. Inbound HTTP adapters. To add a module: construct its handler here and append it to Routes below.
	routes := httpAdapter.Routes{
		API: []httpAdapter.Registrar{
			handler.NewMeHandler(getMe).WithActAs(getActAsState),
			handler.NewDepartmentHandler(createDepartment, updateDepartment, deleteDepartment, listDepartments),
			handler.NewCurriculumHandler(createCurriculum, updateCurriculum, deleteCurriculum, listCurricula),
			handler.NewUserHandler(createUser, updateUser, deleteUser, listUsers),
			handler.NewOfferingHandler(listOfferings, listOfferingYears, createOffering, updateOffering, deleteOfferings, listAccess, setAccess, removeAccess),
			handler.NewImportHandler(previewImport, applyImport, maxImportBytes),
			handler.NewCoverHandler(putCover, getCover, deleteCover, maxCoverBytes),
			handler.NewSubmissionHandler(uploadSubmission, getSubmission, listSubmissions, getPortfolioPDF, listPortfolios, downloadPortfolios, cfg.MaxUploadMB),
			handler.NewTemplateHandler(templateService),
			handler.NewConfigHandler(handler.AppConfig{
				MaxUploadMB: cfg.MaxUploadMB, MaxCoverMB: cfg.MaxCoverMB, MaxImportMB: cfg.MaxImportMB,
				BuildTimeoutMinutes: cfg.BuildTimeoutMinutes,
			}),
		},
		Internal: []httpAdapter.Registrar{
			handler.NewInternalAuthHandler(loginSync),
		},
	}
	if actAsEnabled {
		routes.API = append(routes.API, handler.NewActAsHandler(
			queries.NewListActAsUsersHandler(userRepo), commands.NewStartActAsHandler(userRepo), cfg.CookieSecure))
	}
	router := httpAdapter.SetupRouter(httpAdapter.RouterConfig{
		EnablePprof:      cfg.EnablePprof,
		EnableSwagger:    cfg.EnableSwagger,
		InternalAPIToken: cfg.InternalAPIToken,
		ActorLoader:      userRepo,
		DevActAsEnabled:  actAsEnabled,
	}, handler.NewHealthHandler(store).WithStorage(fileStore).WithQueue(queueMonitor), routes)

	// 6. Serve with graceful shutdown. Uploads can be large, so no write/read deadline on the body.
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	slog.Info("server listening", "addr", srv.Addr)

	// Worker results arrive on the status queue; consume them until shutdown.
	consumerCtx, stopConsumer := context.WithCancel(ctx)
	defer stopConsumer()
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		rabbitmq.NewStatusConsumer(rabbitCfg, processBuildStatus).Run(consumerCtx)
	}()

	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		jobs.Run(consumerCtx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
	}
	slog.Info("shutting down server gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}
	stopConsumer()
	select {
	case <-consumerDone:
	case <-time.After(15 * time.Second):
		slog.Warn("status consumer did not stop in time")
	}
	select {
	case <-schedulerDone:
	case <-time.After(5 * time.Second):
		slog.Warn("background jobs did not stop in time")
	}
	slog.Info("server exited")
	return nil
}

func setupLogger(level string) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(telemetry.NewTraceHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))))
}

// ensureBucket makes sure the bucket exists, retrying for a while so that a slowly starting MinIO does not crash the service.
func ensureBucket(ctx context.Context, s *storage.MinIO) error {
	var err error
	for attempt := 1; attempt <= 15; attempt++ {
		if err = s.EnsureBucket(ctx); err == nil {
			return nil
		}
		slog.Warn("object storage not ready", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("object storage: %w", err)
}
