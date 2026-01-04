FROM golang:1.24.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git curl

# Configure Go proxy for faster downloads
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
RUN go build -o news-portal ./cmd/app

# Stage 2 — Final runtime image
FROM alpine:latest

WORKDIR /root/

# Certs for https requests
RUN apk --no-cache add ca-certificates postgresql-client

# Copy necessary files
COPY --from=builder /app/news-portal ./news-portal
COPY --from=builder /app/docs/patches ./migrations
COPY --from=builder /app/frontend ./frontend
COPY --from=builder /app/config.toml ./config.toml

# Expose port
EXPOSE 3000

# Wait for PostgreSQL, then start app
CMD /bin/sh -c " \
  echo 'Waiting for PostgreSQL...'; \
  until pg_isready -h postgres -p 5432 -U user; do \
    echo 'PostgreSQL is unavailable - sleeping'; \
    sleep 2; \
  done; \
  echo 'PostgreSQL is up - starting app...'; \
  ./news-portal"