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
	"stepik-payments-course/internal/httpapi"
	"stepik-payments-course/internal/payment"
	mockprovider "stepik-payments-course/internal/provider/mock"
	"stepik-payments-course/internal/storage/postgres"
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

	service := payment.NewService(postgres.NewRepository(db), mockprovider.NewProvider())
	app := fiber.New()
	httpapi.NewHandler(service).Register(app)

	go func() {
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Println(err)
			stop()
		}
	}()

	<-ctx.Done()
	_ = app.Shutdown()
}
