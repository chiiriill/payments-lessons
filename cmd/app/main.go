package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"stepik-payments-course/internal/config"
	httpHandler "stepik-payments-course/internal/infrastructure/http/handler"
	paymentsRepo "stepik-payments-course/internal/infrastructure/repository/payments"
	mockProvider "stepik-payments-course/internal/provider/mock"
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
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := paymentsRepo.NewRepo(db)
	provider := mockProvider.NewProvider()

	createPayment := createPaymentUseCase.New(repo, provider)
	getPayment := getPaymentUseCase.New(repo)
	checkPayment := checkPaymentUseCase.New(repo, provider)
	refundPayment := refundPaymentUseCase.New(repo, provider)
	receiveWebhook := receiveWebhookUseCase.New(repo)

	app := fiber.New()
	httpHandler.New(createPayment, getPayment, checkPayment, refundPayment, receiveWebhook).Register(app)

	go func() {
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Println(err)
			stop()
		}
	}()

	<-ctx.Done()
	_ = app.Shutdown()
}
