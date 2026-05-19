package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wa-server/internal/agent"
	"github.com/wa-server/internal/api/handlers"
	"github.com/wa-server/internal/api/webhook"
	"github.com/wa-server/internal/auth"
	"github.com/wa-server/internal/config"
	"github.com/wa-server/internal/metrics"
	"github.com/wa-server/internal/queue"
	"github.com/wa-server/internal/ratelimit"
	"github.com/wa-server/internal/repository"
	"github.com/wa-server/internal/service"
	"github.com/wa-server/internal/whatsapp"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.Info("starting WhatsApp Gateway server...")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := repository.NewPostgresDB(repository.PostgresConfig{
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		User:         cfg.Database.User,
		Password:     cfg.Database.Password,
		Database:     cfg.Database.Database,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxLifetime:  cfg.Database.MaxLifetime,
	})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()
	slog.Info("connected to database")

	rmq, err := queue.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	defer rmq.Close()
	slog.Info("connected to RabbitMQ")

	rdb, err := agent.NewRedisClient(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	defer rdb.Close()
	slog.Info("connected to Redis")

	met := metrics.New()

	bucket := ratelimit.NewTokenBucket(float64(cfg.RateLimit.RequestsPerSecond), cfg.RateLimit.Burst)
	transport := ratelimit.NewTransport(nil, bucket, met)

	msgRepo := repository.NewMessageRepository(db)
	contactRepo := repository.NewContactRepository(db)
	convRepo := repository.NewConversationRepository(db)
	templateRepo := repository.NewTemplateRepo(db)
	companyRepo := repository.NewCompanyRepo(db)
	billingRepo := repository.NewBillingRepository(db)
	phoneNumberRepo := repository.NewPhoneNumberRepository(db)

	publisher := queue.NewPublisher(rmq)

	wsHub := webhook.NewWebSocketHub()
	go wsHub.Run()

	waHandler := webhook.NewWhatsAppHandler(
		cfg,
		msgRepo,
		contactRepo,
		convRepo,
		templateRepo,
		publisher,
		wsHub,
	)

	waClient := whatsapp.NewClient(cfg.WhatsApp.PhoneNumberID, cfg.WhatsApp.WABAID, cfg.WhatsApp.AccessToken, cfg.WhatsApp.APIVersion)
	waClient.HTTPClient.Transport = transport

	templateSvc := service.NewTemplateService(templateRepo, waClient)
	billingSvc := service.NewBillingService(billingRepo, companyRepo, waClient)
	phoneNumberSvc := service.NewPhoneNumberService(phoneNumberRepo, convRepo, waClient)
	pricingRepo := repository.NewPricingRepository(db)
	pricingSvc := service.NewPricingService(pricingRepo, phoneNumberRepo, waClient, cfg.WhatsApp.WABAID)
	outboundHandler := handlers.NewOutboundHandler(msgRepo, publisher, "default")
	templateHandler := handlers.NewTemplateHandler(templateSvc)
	billingHandler := handlers.NewBillingHandler(billingSvc)
	phoneNumberHandler := handlers.NewPhoneNumberHandler(phoneNumberSvc)
	pricingHandler := handlers.NewPricingHandler(pricingSvc)

	jwtAuth := auth.NewJWT(cfg.Auth.JWT.Secret, cfg.Auth.JWT.ExpiryDuration, cfg.Auth.JWT.RefreshDuration)
	userRepo := repository.NewUserRepo(db)
	userSvc := service.NewUserService(userRepo, jwtAuth)
	userHandler := handlers.NewUserHandler(userSvc)
	companySvc := service.NewCompanyService(companyRepo)
	companyHandler := handlers.NewCompanyHandler(companySvc)

	authMW := auth.Middleware(jwtAuth)

	agentTrackers := make([]*agent.Tracker, 5)
	for i := 0; i < 5; i++ {
		agentID := fmt.Sprintf("worker-%d", i)
		agentTrackers[i] = agent.NewTracker(rdb, agentID)
	}

	workerPool := queue.NewWorkerPool(rmq, waClient, msgRepo, contactRepo, companyRepo, billingRepo, convRepo, phoneNumberRepo, 5)
	workerPool.WithMetrics(met).WithAgentTrackers(agentTrackers)
	if err := workerPool.Start(); err != nil {
		return fmt.Errorf("start worker pool: %w", err)
	}
	defer workerPool.Stop()
	slog.Info("worker pool started")

	mux := http.NewServeMux()

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			waHandler.Verify(w, r)
		} else if r.Method == http.MethodPost {
			met.IncMessagesReceived()
			waHandler.HandleWebhook(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/ws", wsHub.HandleWS)
	mux.Handle("/metrics", met.Handler())

	outboundHandler.RegisterRoutes(mux)
	templateHandler.RegisterRoutes(mux)
	billingHandler.RegisterRoutes(mux)
	phoneNumberHandler.RegisterRoutes(mux, authMW)
	pricingHandler.RegisterRoutes(mux)
	userHandler.RegisterRoutes(mux, authMW)
	companyHandler.RegisterRoutes(mux, authMW)

	mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
		agents, err := agent.ListAgents(r.Context(), rdb)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, agents)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.Error("failed to write health response", "error", err)
		}
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	billingCtx, billingCancel := context.WithCancel(context.Background())
	defer billingCancel()
	go func() {
		slog.Info("starting periodic sync", "interval", cfg.Billing.SyncInterval)
		ticker := time.NewTicker(cfg.Billing.SyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-billingCtx.Done():
				return
			case <-ticker.C:
				end := time.Now()
				start := end.Add(-7 * 24 * time.Hour)

				if _, err := phoneNumberSvc.SyncFromMeta(context.Background()); err != nil {
					slog.Error("periodic phone number sync failed", "error", err)
				}

				if _, err := pricingSvc.SyncFromMeta(context.Background()); err != nil {
					slog.Error("periodic pricing sync failed", "error", err)
				}

				if _, err := billingSvc.SyncCostsFromMeta(context.Background(), start, end); err != nil {
					slog.Error("periodic billing sync failed", "error", err)
				}
			}
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "address", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutting down server...", "signal", sig)
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data, _ := json.Marshal(v)
	_, _ = w.Write(data)
}
