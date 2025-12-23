# Makefile for news-portal project

APP_NAME := news-portal
ENV_FILE := ./envs/.env.dev

# Docker
.PHONY: up down restart logs

docker-build:
	docker-compose --env-file $(ENV_FILE) build

docker-up:
	docker-compose --env-file $(ENV_FILE) up


docker-down:
	docker-compose --env-file $(ENV_FILE) down

restart: docker-down docker-up

logs:
	docker-compose --env-file $(ENV_FILE) logs -f

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

# Swagger
.PHONY: swag

swag:
	swag init --generalInfo cmd/app/main.go --output docs
