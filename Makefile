.PHONY: up down restart logs build test lint

up:
	docker compose up --build -d

down:
	docker compose down

down-clean:
	docker compose down -v

restart:
	docker compose restart app worker

logs:
	docker compose logs -f app worker

build:
	go build ./...

test:
	go test ./...

test-integration:
	DATABASE_URL=postgres://postgres:postgres@localhost:5432/payments?sslmode=disable \
	go test ./internal/infrastructure/repository/payments/...
