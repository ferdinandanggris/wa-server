package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wa-server/internal/api/handlers"
	"github.com/wa-server/internal/api/webhook"
	"github.com/wa-server/internal/config"
	"github.com/wa-server/internal/queue"
	"github.com/wa-server/internal/repository"
)

type mockWhatsAppClient struct{}

func (m *mockWhatsAppClient) SendMessage(ctx context.Context, phoneNumberID, to, messageType, content string, mediaURL string) (string, error) {
	slog.Info("mock send message", "phoneNumberID", phoneNumberID, "to", to, "type", messageType, "content", content)
	return "mock_message_id_" + fmt.Sprint(time.Now().Unix()), nil
}

func (m *mockWhatsAppClient) SendTemplateMessage(ctx context.Context, phoneNumberID, to, templateID string, params map[string]string) (string, error) {
	slog.Info("mock send template", "phoneNumberID", phoneNumberID, "to", to, "templateID", templateID, "params", params)
	return "mock_message_id_" + fmt.Sprint(time.Now().Unix()), nil
}

func main() {
	slog.Info("starting WhatsApp Gateway server...")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
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
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to database")

	rmq, err := queue.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rmq.Close()
	slog.Info("connected to RabbitMQ")

	msgRepo := repository.NewMessageRepository(db)
	contactRepo := repository.NewContactRepository(db)
	convRepo := repository.NewConversationRepository(db)

	publisher := queue.NewPublisher(rmq)

	wsHub := webhook.NewWebSocketHub()
	go wsHub.Run()

	waHandler := webhook.NewWhatsAppHandler(
		cfg,
		msgRepo,
		contactRepo,
		convRepo,
		publisher,
		wsHub,
	)

	outboundHandler := handlers.NewOutboundHandler(msgRepo, publisher, "default")

	workerPool := queue.NewWorkerPool(rmq, &mockWhatsAppClient{}, msgRepo, 5)
	if err := workerPool.Start(); err != nil {
		slog.Error("failed to start worker pool", "error", err)
		os.Exit(1)
	}
	slog.Info("worker pool started")
	defer workerPool.Stop()

	mux := http.NewServeMux()

	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			waHandler.Verify(w, r)
		} else if r.Method == http.MethodPost {
			waHandler.HandleWebhook(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/ws", wsHub.HandleWS)

	outboundHandler.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		slog.Info("server starting", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}
