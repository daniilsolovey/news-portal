# Makefile for news-portal project

APP_NAME := news-portal

# Docker
.PHONY: up down restart logs

docker-build:
	docker-compose build

docker-up:
	docker-compose up


docker-down:
	docker-compose down

restart: docker-down docker-up

logs:
	docker-compose logs -f

# Go
.PHONY: build run tidy test fmt vet

build:
	go build -o bin/$(APP_NAME) ./cmd/app

run:
	go run ./cmd/app/main.go

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./... -v

# Run integration tests (requires test database to be running)
test-integration:
	go test -v ./internal/db/...

# Test Database (PostgreSQL)
.PHONY: test-db-up test-db-down test-db-remove test-db-restart

# Start test PostgreSQL container
test-db-up:
	docker-compose -f docker-compose.test.yml up -d

# Stop test PostgreSQL container
test-db-down:
	docker-compose -f docker-compose.test.yml down

# Remove test PostgreSQL container and volumes
test-db-remove:
	docker-compose -f docker-compose.test.yml down -v

# Restart test PostgreSQL container
test-db-restart: test-db-down test-db-up

# Swagger
.PHONY: swag

swag:
	swag init --generalInfo cmd/app/main.go --output docs


genna:
	genna model -c "postgres://user:password@localhost:5432/news_portal?sslmode=disable" -o internal/db/model.go -t "public.*" -f

mfd-xml:
	@mfd-generator xml -c "postgres://user:password@localhost:5432/news_portal?sslmode=disable" -m ./docs/model/newsportal.mfd

mfd-model:
	@mfd-generator model -m ./docs/model/newsportal.mfd -p db -o ./internal/db

mfd-repo:
	@mfd-generator repo -m ./docs/model/newsportal.mfd -p db -o ./internal/db
