# News Portal

A Go-based news portal application with clean architecture, RPC and REST API support, and PostgreSQL database.

## 🎥 Demo

![Demo](assets/images/news-portal-demo.png)

## ⚙️ Architecture

```
                ┌────────────┐
                │   Client   │
                │  (Browser) │
                └─────┬──────┘
                      │ HTTP (Echo)
                ┌─────▼──────┐
                │    App     │  ← HTTP server, Static files, RPC
                └─────┬──────┘
                      │
                ┌─────▼──────┐
                │  Manager   │  ← Business logic (newsportal)
                └─────┬──────┘
                      │
            ┌─────────▼─────────┐
            │     Repository    │  ← Database access (go-pg)
            └────────┬──────────┘
                     │
       ┌─────────────▼─────────────┐
       │ PostgreSQL (storage)      │
       └───────────────────────────┘
```

## 📁 Project Structure

```
.
├── cmd/app              # Entry-point and application initialization
├── internal
│   ├── app              # Application setup and routing
│   ├── db               # Database layer (go-pg models, repositories)
│   ├── newsportal       # Business logic layer (Manager)
│   ├── rest             # REST API handlers (available, not active)
│   └── rpc              # RPC handlers (zenrpc)
├── frontend             # Frontend web interface
│   ├── index.html       # Main HTML page
│   ├── app.js           # JavaScript application logic
│   └── styles.css       # CSS styles
├── docs                 # Documentation and migrations
│   ├── patches          # Database migrations (goose)
│   └── swagger.*        # Swagger documentation (generated)
├── config.toml          # Application configuration (TOML)
├── Makefile             # Build/run commands
└── docker-compose.yml   # Docker infrastructure
```

## 🚀 Quick Start

Make sure you have `Docker`, `Make`, and `Go` installed.

### Start the service

```bash
make docker-up
```

The application uses `config.toml` for configuration.

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

**Note**: Make sure PostgreSQL is running and accessible. Update `config.toml` with your database connection settings.

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

### Run integration tests

```bash
make test-db-up          # Start test database
make test-integration    # Run integration tests
make test-db-down        # Stop test database
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

The frontend is automatically served by the Echo router at the root path (`/`) and static files are available at `/static/`.

## 🔌 API Endpoints

### RPC API (Active)

The service provides a JSON-RPC 2.0 API using [zenrpc](https://github.com/vmkteam/zenrpc):

- `POST /rpc` - JSON-RPC endpoint
- `GET /doc/*` - Service Method Discovery (SMD) documentation

**Available RPC Methods:**
- `news.List(filter)` - Get all news with optional filtering by tagId and categoryId, with pagination
- `news.Count(filter)` - Get total count of news items
- `news.ByID(id)` - Get news item by ID with full content
- `news.Categories()` - Get all categories
- `news.Tags()` - Get all tags

### REST API (Available but not active)

REST API handlers are implemented in `internal/rest` but currently commented out in `app.go`. To enable:

1. Uncomment the REST handler initialization in `internal/app/app.go`
2. Register REST routes in the Echo router

**Available REST Endpoints** (when enabled):
- `GET /api/v1/news` - Get all news with optional filtering
- `GET /api/v1/news/count` - Get total count of news items
- `GET /api/v1/news/:id` - Get news item by ID
- `GET /api/v1/categories` - Get all categories
- `GET /api/v1/tags` - Get all tags
- `GET /health` - Health check endpoint

### Static Files

- `GET /` - Frontend web interface
- `GET /static/*` - Static frontend files (CSS, JS)

## 📚 Swagger Documentation

Swagger documentation can be generated (but is not currently integrated in routes):

```bash
make swag
```

The generated documentation is available in `docs/swagger.json` and `docs/swagger.yaml`.

## 📝 Example Makefile Commands

```bash
make docker-up         # start containers
make docker-down       # stop containers
make restart           # restart containers
make logs              # view logs
make swag              # generate Swagger
make build             # build Go binary
make run               # run application locally
make test              # run tests
make test-integration  # run integration tests
make test-db-up        # start test database
make test-db-down      # stop test database
```

## 🏗 Features

- **Clean Architecture**: Separation of concerns with business logic, repository, and API layers
- **Database Support**: PostgreSQL with go-pg ORM and connection pooling
- **RPC API**: JSON-RPC 2.0 API with zenrpc framework
- **REST API**: RESTful API handlers (available, can be enabled)
- **Frontend Interface**: Modern web-based UI for API interaction
- **Static File Serving**: Built-in static file server for frontend assets
- **Graceful Shutdown**: Graceful shutdown with 5-second timeout for HTTP server
- **Configuration**: TOML-based configuration management
- **Migrations**: Database migration support with goose
- **Logging**: Structured logging with slog

## 🔧 Configuration

Configuration is managed via TOML file (`config.toml`). The default configuration file is `config.toml` in the project root.

### Configuration Structure

```toml
[Database]
Addr     = "postgres:5432"
User     = "user"
Database = "news_portal"
Password = "password"
PoolSize = 5

[App]
Host = "0.0.0.0"
Port = 3000
```

### Command Line Options

- `-config` - Path to TOML configuration file (default: `config.toml`)
- `-debug` - Enable debug mode for logging

Example:

```bash
./news-portal -config config.toml -debug
```

## 🗄 Database Migrations

The project uses [goose](https://github.com/pressly/goose) for database migrations. Migrations are located in the `docs/patches/` directory.

### Automatic Migrations (Docker)

When running with Docker (`make docker-up`), the application waits for PostgreSQL to be ready before starting. Migrations should be run manually or integrated into your deployment process.

### Manual Migrations

To run migrations manually, you need to have goose installed:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then run migrations:

```bash
goose -dir ./docs/patches postgres "postgres://user:password@localhost:5432/news_portal?sslmode=disable" up
```

## 📦 Adding New Features

1. **Database Models**: Add to `internal/db/` (use genna or mfd-generator for model generation)
2. **Business Logic**: Implement in `internal/newsportal/`
3. **RPC Handlers**: Add to `internal/rpc/`
4. **REST Handlers**: Add to `internal/rest/` (and enable in `internal/app/app.go`)
5. **Database Operations**: Implement in `internal/db/` repositories

## 🔧 Code Generation

The project uses code generation tools:

### Model Generation

- **genna**: Generate Go models from PostgreSQL schema
  ```bash
  make genna
  ```

- **mfd-generator**: Generate models and repositories from MFD files
  ```bash
  make mfd-model  # Generate models
  make mfd-repo   # Generate repositories
  make mfd-xml    # Generate XML from database
  ```

### RPC Code Generation

RPC service code is generated using zenrpc:

```bash
cd internal/rpc && go generate
```

---
