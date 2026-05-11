package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"stepik-payments-course/internal/config"
	providerClient "stepik-payments-course/internal/infrastructure/provider/mockclient"
	paymentsRepo "stepik-payments-course/internal/infrastructure/repository/payments"
	"stepik-payments-course/internal/usecase/processStalePaymentsUseCase"
)

const (
	pollInterval  = 30 * time.Second
	pollTimeout   = 25 * time.Second
	staleDuration = 5 * time.Minute
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.Load()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := paymentsRepo.NewRepo(db)
	provider := providerClient.New(cfg.ProviderBaseURL, cfg.ProviderTimeout, cfg.ProviderRetryCount)
	uc := processStalePaymentsUseCase.New(repo, provider, logger, staleDuration)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	logger.InfoContext(ctx, "stale payments worker started",
		"poll_interval", pollInterval,
		"stale_duration", staleDuration,
	)

	go func() {
		<-ctx.Done()
		logger.Info("shutdown signal received, waiting for current poll to finish")
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("stale payments worker stopped")
			return
		case <-ticker.C:
			runPoll(ctx, uc, logger)
		}
	}
}

func runPoll(ctx context.Context, uc *processStalePaymentsUseCase.UseCase, logger *slog.Logger) {
	tickCtx, cancel := context.WithTimeout(context.Background(), pollTimeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, "poll panicked", "panic", r)
		}
	}()

	uc.Execute(tickCtx)
}
