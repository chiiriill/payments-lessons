package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"stepik-payments-course/internal/config"
	httpHandler "stepik-payments-course/internal/infrastructure/http/handler"
	httpMiddleware "stepik-payments-course/internal/infrastructure/http/middleware"
	providerClient "stepik-payments-course/internal/infrastructure/provider/mockclient"
	paymentsRepo "stepik-payments-course/internal/infrastructure/repository/payments"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/getPaymentUseCase"
	"stepik-payments-course/internal/usecase/receiveWebhookUseCase"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With("component", "app")
	slog.SetDefault(logger)

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := paymentsRepo.NewRepo(db)
	provider := providerClient.New(cfg.ProviderBaseURL, cfg.ProviderTimeout, cfg.ProviderRetryCount)

	createPayment := createPaymentUseCase.New(repo, provider, logger)
	getPayment := getPaymentUseCase.New(repo)
	checkPayment := checkPaymentUseCase.New(repo, provider)
	refundPayment := refundPaymentUseCase.New(repo, provider)
	receiveWebhook := receiveWebhookUseCase.New(repo, logger)

	app := fiber.New(fiber.Config{AppName: cfg.AppName})
	app.Use(httpMiddleware.Metrics())
	app.Use(httpMiddleware.RequestLogger(logger))
	httpHandler.New(createPayment, getPayment, checkPayment, refundPayment, receiveWebhook, cfg.WebhookSecret, db.Ping).Register(app)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/debug/pprof/", httppprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", httppprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", httppprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", httppprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", httppprof.Trace)
		logger.Info("admin server started", "addr", cfg.MetricsAddr)
		if err := http.ListenAndServe(cfg.MetricsAddr, mux); err != nil {
			logger.Error("metrics server error", "error", err)
		}
	}()

	logger.Info("starting", "app", cfg.AppName, "addr", cfg.HTTPAddr)
	go func() {
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			logger.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	if err := app.ShutdownWithTimeout(cfg.ShutdownTimeout); err != nil {
		logger.Error("shutdown timeout exceeded", "error", err)
	}
	logger.Info("shutdown complete")
}
