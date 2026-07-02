package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plugin-execution-system/internal/config"
	"plugin-execution-system/internal/event"
	"plugin-execution-system/internal/handler"
	"plugin-execution-system/internal/logging"
	"plugin-execution-system/internal/queue"
	"plugin-execution-system/internal/repository"
	"plugin-execution-system/internal/router"
	"plugin-execution-system/internal/service"
	"plugin-execution-system/internal/storage"
	pgstore "plugin-execution-system/internal/storage/postgres"
	"plugin-execution-system/internal/worker"
)

type stopper interface{ Stop() }

type multiStopper []stopper

func (m multiStopper) Stop() {
	for i := len(m) - 1; i >= 0; i-- {
		if m[i] != nil {
			m[i].Stop()
		}
	}
}

type App struct {
	server  *http.Server
	stopper stopper
	cleanup func(context.Context) error
}

func main() {
	app, err := buildApp()
	if err != nil {
		logging.Error("startup_failed", logging.Fields{"error": err.Error()})
		os.Exit(1)
	}
	go func() {
		logging.Info("server_listening", logging.Fields{"addr": app.server.Addr})
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Error("server_failed", logging.Fields{"error": err.Error()})
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if app.stopper != nil {
		app.stopper.Stop()
	}
	if app.cleanup != nil {
		_ = app.cleanup(ctx)
	}
	_ = app.server.Shutdown(ctx)
}

func buildApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	switch cfg.Storage.Driver {
	case "local-json":
		return buildLocalJSONApp(cfg)
	case "postgres":
		return buildPostgresApp(cfg)
	default:
		return nil, fmt.Errorf("unsupported metadata store: %s", cfg.Storage.Driver)
	}
}

// buildPostgresApp is intentionally fail-closed unless a real database/sql
// Postgres driver is registered and POSTGRES_DSN is configured. v9 moves the
// process away from implicit local fallbacks; local-json is now a separate dev
// mode, never a production fallback.
func buildPostgresApp(cfg config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgstore.OpenDB(ctx, pgstore.OpenOptions{
		DriverName:      cfg.Storage.PostgresDriver,
		DSN:             cfg.Storage.PostgresDSN,
		MaxOpenConns:    cfg.Storage.MaxOpenConns,
		MaxIdleConns:    cfg.Storage.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Storage.ConnMaxLifetime) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if cfg.Storage.AutoMigrate {
		if err := pgstore.Migrate(ctx, db, cfg.Storage.MigrationPath); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	repos := repository.NewPostgresRepositories(db)
	authSvc := service.NewAuthService(repos.Auth)
	auditSvc := service.NewAuditService(repos.Audit)
	eventSvc := service.NewExecutionEventService(repos.Event, event.NewBus())
	attemptSvc := service.NewExecutionAttemptService(repos.Attempt)
	webhookSvc := service.NewWebhookService(repos.Webhook)
	webhookRetry := worker.NewWebhookRetryScheduler(webhookSvc, worker.WebhookRetryOptions{Interval: 30 * time.Second, Batch: 50})
	webhookRetry.Start(context.Background())
	eventSvc.Subscribe("*", webhookSvc.HandleExecutionEvent)
	pluginSM := service.NewPluginStateMachine()
	pluginSvc := service.NewPluginService(repos.Plugin, pluginSM)
	manifestSvc := service.NewManifestService(cfg.Security.TrustedPluginPublicKeys...)
	registrySvc := service.NewRegistryService(manifestSvc, pluginSvc, repos.Registry, repos.Plugin)
	if _, err := registrySvc.ReloadPlugins(cfg.Plugin.Dir); err != nil {
		logging.Warn("plugin_reload_failed", logging.Fields{"error": err.Error()})
	}
	meta := pgstore.NewMetadataStore(db)
	execSM := service.NewExecutionStateMachine()
	idemSvc := service.NewIdempotencyService(repos.Execution)
	execSvc := service.NewEnterpriseExecutionService(meta, repos.Execution, pluginSvc, idemSvc, execSM)
	resultSvc := service.NewResultService(repos.Result)
	runtimeSvc := service.NewRuntimeService(service.NewProcessRunner(cfg.Execution.AllowedCommands...), cfg.Execution.MaxOutputBytes).WithContainerRunner(service.NewContainerRunner(cfg.Execution.ContainerRuntimeEnabled))
	pgQueue := queue.NewPostgresQueue(db)
	workerID := os.Getenv("PES_WORKER_ID")
	if workerID == "" {
		workerID = "postgres-worker"
	}
	durableHandler := worker.NewMetadataExecutionHandler(meta, repos.Execution, pluginSvc, runtimeSvc, resultSvc, workerID)
	durablePool := worker.NewDurableWorkerPool(pgQueue, durableHandler, worker.DurableWorkerOptions{
		WorkerID:          workerID,
		WorkerCount:       cfg.Execution.WorkerCount,
		LeaseDuration:     time.Duration(cfg.Execution.LeaseDurationSeconds) * time.Second,
		HeartbeatInterval: time.Duration(cfg.Execution.HeartbeatSeconds) * time.Second,
		MaxAttempts:       cfg.Execution.MaxAttempts,
	})
	if err := durablePool.Start(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	r := router.NewRouter(router.Handlers{
		Auth:      handler.NewAuthHandler(authSvc),
		Plugin:    handler.NewPluginHandler(pluginSvc, registrySvc, cfg.Plugin.Dir, auditSvc),
		Execution: handler.NewExecutionHandler(execSvc),
		Result:    handler.NewResultHandler(resultSvc, execSvc),
		Observe:   handler.NewExecutionObserveHandler(execSvc, eventSvc, attemptSvc),
		Audit:     handler.NewAuditHandler(auditSvc),
		Health:    handler.NewHealthHandler(cfg.Plugin.Dir),
		Webhook:   handler.NewWebhookHandler(webhookSvc),
		AuthSvc:   authSvc,
	})
	return &App{server: &http.Server{Addr: cfg.Server.Addr, Handler: r}, stopper: multiStopper{webhookRetry, durablePool}, cleanup: func(ctx context.Context) error { return db.Close() }}, nil
}

func buildLocalJSONApp(cfg config.Config) (*App, error) {
	store, err := storage.OpenJSONStore(cfg.Storage.Dir)
	if err != nil {
		return nil, err
	}
	if err := storage.Migrate(store); err != nil {
		return nil, err
	}
	if err := storage.SeedDefaultUsers(store, cfg.Auth.DemoToken, cfg.Auth.AdminToken); err != nil {
		return nil, err
	}
	repos := repository.NewRepositories(store)
	authSvc := service.NewAuthService(repos.Auth)
	auditSvc := service.NewAuditService(repos.Audit)
	eventSvc := service.NewExecutionEventService(repos.Event, event.NewBus())
	attemptSvc := service.NewExecutionAttemptService(repos.Attempt)
	webhookSvc := service.NewWebhookService(repos.Webhook)
	webhookRetry := worker.NewWebhookRetryScheduler(webhookSvc, worker.WebhookRetryOptions{Interval: 30 * time.Second, Batch: 50})
	webhookRetry.Start(context.Background())
	eventSvc.Subscribe("*", webhookSvc.HandleExecutionEvent)
	pluginSM := service.NewPluginStateMachine()
	pluginSvc := service.NewPluginService(repos.Plugin, pluginSM)
	manifestSvc := service.NewManifestService(cfg.Security.TrustedPluginPublicKeys...)
	registrySvc := service.NewRegistryService(manifestSvc, pluginSvc, repos.Registry, repos.Plugin)
	_, _ = registrySvc.ReloadPlugins(cfg.Plugin.Dir)
	queue := worker.NewExecutionQueue(cfg.Execution.QueueSize)
	execSM := service.NewExecutionStateMachine()
	idemSvc := service.NewIdempotencyService(repos.Execution)
	var execSvc *service.ExecutionService
	execSvc = service.NewExecutionService(repos.Execution, pluginSvc, idemSvc, execSM, auditSvc, queue).WithEventService(eventSvc)
	resultSvc := service.NewResultService(repos.Result)
	runtimeSvc := service.NewRuntimeService(service.NewProcessRunner(cfg.Execution.AllowedCommands...), cfg.Execution.MaxOutputBytes).WithContainerRunner(service.NewContainerRunner(cfg.Execution.ContainerRuntimeEnabled))
	execWorker := worker.NewExecutionWorkerWithEvents(execSvc, pluginSvc, runtimeSvc, resultSvc, auditSvc, eventSvc, attemptSvc)
	pool := worker.NewWorkerPool(queue, execWorker, cfg.Execution.WorkerCount)
	pool.Start(context.Background())
	if recovered, err := execSvc.RecoverIncompleteExecutions(context.Background(), "startup-recovery"); err != nil {
		logging.Warn("startup_recovery_failed", logging.Fields{"error": err.Error()})
	} else if recovered > 0 {
		logging.Info("startup_recovered_executions", logging.Fields{"count": recovered})
	}
	r := router.NewRouter(router.Handlers{
		Auth:      handler.NewAuthHandler(authSvc),
		Plugin:    handler.NewPluginHandler(pluginSvc, registrySvc, cfg.Plugin.Dir, auditSvc),
		Execution: handler.NewExecutionHandler(execSvc),
		Result:    handler.NewResultHandler(resultSvc, execSvc),
		Observe:   handler.NewExecutionObserveHandler(execSvc, eventSvc, attemptSvc),
		Audit:     handler.NewAuditHandler(auditSvc),
		Health:    handler.NewHealthHandler(cfg.Plugin.Dir),
		Webhook:   handler.NewWebhookHandler(webhookSvc),
		AuthSvc:   authSvc,
	})
	return &App{server: &http.Server{Addr: cfg.Server.Addr, Handler: r}, stopper: multiStopper{webhookRetry, pool}}, nil
}
