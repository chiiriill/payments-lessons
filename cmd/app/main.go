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
	mockProvider "stepik-payments-course/internal/infrastructure/provider/mock"
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

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := paymentsRepo.NewRepo(db)
	provider := mockProvider.NewProvider(cfg.ProviderBaseURL, cfg.ProviderTimeout)

	createPayment := createPaymentUseCase.New(repo, provider)
	getPayment := getPaymentUseCase.New(repo)
	checkPayment := checkPaymentUseCase.New(repo, provider)
	refundPayment := refundPaymentUseCase.New(repo, provider)
	receiveWebhook := receiveWebhookUseCase.New(repo)

	app := fiber.New(fiber.Config{AppName: cfg.AppName})
	httpHandler.New(createPayment, getPayment, checkPayment, refundPayment, receiveWebhook).Register(app)

	log.Printf("%s listening on %s", cfg.AppName, cfg.HTTPAddr)
	go func() {
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Printf("server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	if err := app.ShutdownWithTimeout(cfg.ShutdownTimeout); err != nil {
		log.Printf("shutdown timeout exceeded: %v", err)
	}
	log.Println("shutdown complete")
}
