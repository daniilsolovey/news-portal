# News Portal Template

A Go-based microservice template with clean architecture, dependency injection (Wire), and PostgreSQL support.

## 🎥 Demo

![Demo](assets/images/news-portal-demo.png)

## ⚙️ Architecture

```
                ┌────────────┐
                │   Client   │
                │  (Browser) │
                └─────┬──────┘
                      │ HTTP (Gin)
                ┌─────▼──────┐
                │  Delivery  │  ← HTTP handlers, Static files, Swagger
                └─────┬──────┘
                      │
                ┌─────▼──────┐
                │  UseCase   │  ← Business logic
                └─────┬──────┘
                      │
            ┌─────────▼─────────┐
            │     Repository    │  ← PostgreSQL access
            └────────┬──────────┘
                     │
       ┌─────────────▼─────────────┐
       │ PostgreSQL (storage)      │
       └───────────────────────────┘
```

## 📁 Project Structure

```
.
├── cmd/app              # Entry-point and dependency initialization (Wire)
│   └── wire            # Dependency injection setup
├── configs              # Application configuration (Viper)
├── internal
│   ├── delivery         # HTTP endpoints (Gin handlers)
│   ├── usecase          # Business logic layer
│   ├── domain           # Domain models and conversions
│   └── repository       # Data access layer
│       └── postgres     # PostgreSQL implementation
├── frontend             # Frontend web interface
│   ├── index.html       # Main HTML page
│   ├── app.js           # JavaScript application logic
│   └── styles.css       # CSS styles
├── docs                 # Swagger documentation
├── envs                 # .env files
├── migrations           # Database migrations
├── Makefile             # Build/run commands
└── docker-compose.yml   # Docker infrastructure
```

## 🚀 Quick Start

Make sure you have `Docker`, `Make`, `Go`, and `Swag` installed.

### Start the service

```bash
make docker-up
```

The default environment file is `./envs/.env.dev`.

### Stop the service

```bash
make docker-down
```

### Restart

```bash
make restart
```

## 🛠 Development

### Build the binary

```bash
make build
```

### Run the app locally

```bash
make run
```

### Format, validate and manage dependencies

```bash
make fmt     # go fmt ./...
make vet     # go vet ./...
make tidy    # go mod tidy
```

### Run tests

```bash
make test
```

### Regenerate Wire dependencies

```bash
cd cmd/app/wire && go generate
```

## 📚 Swagger Documentation

To generate:

```bash
make swag
```

Available after startup at:

```
http://localhost:3000/swagger/index.html
```

## 🖥️ Frontend Interface

The project includes a modern web-based frontend interface for interacting with the API. The frontend is served as static files from the same server.

### Access the Frontend

After starting the service, access the frontend at:

```
http://localhost:3000/
```

### Frontend Features

The frontend provides a user-friendly interface for:

- **📰 Get All News** - Browse news with filtering by tags and categories, pagination support
- **📊 News Count** - View total count of news items with optional filters
- **📄 News Details** - View full news article by ID with complete content
- **📁 Categories** - Browse all available news categories
- **🏷️ Tags** - View all available tags

The interface features:
- Modern, responsive design with gradient styling
- Real-time API interaction
- Formatted JSON responses with syntax highlighting
- Error handling and loading states
- Filtering and pagination controls

### Frontend Structure

- `frontend/index.html` - Main HTML structure
- `frontend/app.js` - JavaScript logic for API calls and UI updates
- `frontend/styles.css` - Modern CSS styling with gradients and animations

The frontend is automatically served by the Gin router at the root path (`/`) and static files are available at `/static/`.

## 🔌 API Endpoints

The service provides the following REST API endpoints:

- `GET /` - Frontend web interface
- `GET /static/*` - Static frontend files (CSS, JS)
- `GET /api/v1/all_news` - Get all news with optional filtering by tagId and categoryId, with pagination
- `GET /api/v1/count` - Get total count of news items
- `GET /api/v1/news/:id` - Get news item by ID with full content
- `GET /api/v1/categories` - Get all categories
- `GET /api/v1/tags` - Get all tags
- `GET /health` - Health check endpoint
- `GET /swagger/*any` - Swagger documentation UI

## 📝 Example Makefile Commands

```bash
make docker-up    # start containers
make docker-down  # stop containers
make logs         # view logs
make swag         # generate Swagger
make build        # build Go binary
make run          # run application locally
make test         # run tests
```

## 🏗 Template Features

- **Clean Architecture**: Separation of concerns with delivery, usecase, and repository layers
- **Dependency Injection**: Google Wire for compile-time DI
- **Database Support**: PostgreSQL with connection pooling
- **API**: REST API with Gin framework
- **Frontend Interface**: Modern web-based UI for API interaction
- **Static File Serving**: Built-in static file server for frontend assets
- **Graceful Shutdown**: Graceful shutdown with 5-second timeout for HTTP server
- **Documentation**: Swagger/OpenAPI documentation
- **Configuration**: Viper for configuration management
- **Migrations**: Database migration support

## 🔧 Configuration

Configuration is managed via environment variables or `.env` files. Default values are set in `configs/config.go`.

Key environment variables:
- `HTTP_PORT` - HTTP server port (default: 3000)
- `DATABASE_URL` - PostgreSQL connection string
- `DB_MAX_CONNS` - Maximum number of database connections (default: 5)
- `DB_MAX_CONN_LIFETIME` - Maximum connection lifetime (default: 300s)

## 🗄 Database Migrations

The project uses [goose](https://github.com/pressly/goose) for database migrations. Migrations are located in the `migrations/` directory.

### Automatic Migrations (Docker)

When running with Docker (`make docker-up`), migrations are automatically executed before the application starts. The Dockerfile includes goose and runs migrations on container startup.

### Manual Migrations

To run migrations manually, you need to have goose installed:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then run migrations:

```bash
goose -dir ./migrations postgres "postgres://user:password@localhost:5432/news_portal?sslmode=disable" up
```

## 🔌 Dependency Injection (Wire)

This project uses [Google Wire](https://github.com/google/wire) for dependency injection at compile time. All application components are initialized through Wire providers.

### Initialization Flow

Wire automatically resolves and initializes dependencies in the following order:

1. **Logger** (`ProvideLogger`) - Creates structured logger
2. **PostgreSQL Repository** (`ProvidePostgres`) - Initializes database connection pool
3. **Repository** (`ProvideRepository`) - Creates repository interface wrapper
4. **UseCase** (`ProvideUseCase`) - Initializes business logic layer with repository and logger
5. **Handler** (`ProvideHandler`) - Creates HTTP handlers with usecase and logger
6. **Engine** (`ProvideEngine`) - Creates Gin router with all registered routes

### Service Structure

The `wire.Service` struct contains all initialized components:

```go
type Service struct {
    Postgres *postgres.Repository
    Logger   *slog.Logger
    Engine   *gin.Engine
}
```

### Adding New Providers

To add new dependencies via Wire:

1. Create a provider function in `cmd/app/wire/providers.go`
2. Add it to `wire.Build()` in `cmd/app/wire/wire.go`
3. Regenerate Wire code: `cd cmd/app/wire && go generate`

## 📦 Adding New Features

1. **Domain Models**: Add to `internal/domain/`
2. **Business Logic**: Implement in `internal/usecase/`
3. **HTTP Handlers**: Add to `internal/delivery/`
4. **Database Operations**: Implement in `internal/repository/postgres/`
5. **Routes**: Register in your handler's `RegisterRoutes()` method

---