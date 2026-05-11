.PHONY: up down down-clean build logs test psql lint

up:
	docker compose up --build -d

down:
	docker compose down

down-clean:
	docker compose down -v

build:
	docker compose build

logs:
	docker compose logs -f

test:
	go test ./...

psql:
	docker compose exec postgres psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-payments}

lint:
	go vet ./...
