FROM golang:1.24.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git curl

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Install goose migration tool
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Build
RUN go build -o news-portal ./cmd/app

# Stage 2 — Final runtime image
FROM alpine:latest

WORKDIR /root/

# Certs for https requests
RUN apk --no-cache add ca-certificates postgresql-client

# Copy necessary files
COPY --from=builder /app/news-portal ./news-portal
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/migrations ./migrations

# ENV
ENV DATABASE_URL="postgres://user:password@postgres:5432/news_portal?sslmode=disable"
ENV HTTP_PORT=3000

# Expose port
EXPOSE 3000

# Run migrations, then start app
CMD /bin/sh -c '\
  echo "Waiting for PostgreSQL..."; \
  until pg_isready -h postgres -p 5432 -U user; do \
    echo "PostgreSQL is unavailable - sleeping"; \
    sleep 2; \
  done; \
  echo "PostgreSQL is up - running migrations..."; \
  goose -dir ./migrations postgres "$$DATABASE_URL" up || exit 1; \
  echo "Migrations completed. Starting app..."; \
  ./news-portal'