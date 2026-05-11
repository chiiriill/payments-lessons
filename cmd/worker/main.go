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
	"stepik-payments-course/internal/events"
	providerClient "stepik-payments-course/internal/infrastructure/provider/mockclient"
	paymentsRepo "stepik-payments-course/internal/infrastructure/repository/payments"
	"stepik-payments-course/internal/usecase/processOutboxEventsUseCase"
	"stepik-payments-course/internal/usecase/processStalePaymentsUseCase"
)

const (
	pollInterval   = 30 * time.Second
	pollTimeout    = 25 * time.Second
	staleDuration  = 5 * time.Minute
	outboxInterval = 5 * time.Second
	outboxTimeout  = 4 * time.Second
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
	eventPublisher := events.NewLogPublisher(logger)

	stalePaymentsUC := processStalePaymentsUseCase.New(repo, provider, logger, staleDuration)
	outboxUC := processOutboxEventsUseCase.New(repo, eventPublisher, logger)

	staleTicker := time.NewTicker(pollInterval)
	defer staleTicker.Stop()
	outboxTicker := time.NewTicker(outboxInterval)
	defer outboxTicker.Stop()

	logger.InfoContext(ctx, "worker started",
		"poll_interval", pollInterval,
		"stale_duration", staleDuration,
		"outbox_interval", outboxInterval,
	)

	go func() {
		<-ctx.Done()
		logger.Info("shutdown signal received, waiting for current poll to finish")
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return
		case <-staleTicker.C:
			runPoll(ctx, stalePaymentsUC, logger)
		case <-outboxTicker.C:
			runOutboxPoll(ctx, outboxUC, logger)
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

func runOutboxPoll(ctx context.Context, uc *processOutboxEventsUseCase.UseCase, logger *slog.Logger) {
	tickCtx, cancel := context.WithTimeout(context.Background(), outboxTimeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, "outbox poll panicked", "panic", r)
		}
	}()

	uc.Execute(tickCtx)
}
