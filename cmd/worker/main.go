package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stepik-payments-course/internal/config"
	providerClient "stepik-payments-course/internal/infrastructure/provider/mockclient"
	paymentsRepo "stepik-payments-course/internal/infrastructure/repository/payments"
	"stepik-payments-course/internal/usecase/processStalePaymentsUseCase"

	"github.com/jackc/pgx/v5/pgxpool"
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
	uc := processStalePaymentsUseCase.New(repo, provider, logger)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger.InfoContext(ctx, "stale payments worker started")
	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "stale payments worker stopped")
			return
		case <-ticker.C:
			uc.Execute(ctx)
		}
	}
}
