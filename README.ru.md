# News Portal Template

Шаблон Go-микросервиса с чистой архитектурой, dependency injection (Wire) и PostgreSQL.

## ⚙️ Архитектура

```
                ┌────────────┐
                │   Client   │
                └─────┬──────┘
                      │ HTTP (Gin)
                ┌─────▼──────┐
                │  Delivery  │  ← HTTP-обработчики, Swagger
                └─────┬──────┘
                      │
                ┌─────▼──────┐
                │  UseCase   │  ← Бизнес-логика
                └─────┬──────┘
                      │
            ┌─────────▼─────────┐
            │     Repository    │  ← Работа с PostgreSQL
            └────────┬──────────┘
                     │
       ┌─────────────▼─────────────┐
       │ PostgreSQL (хранилище)    │
       └───────────────────────────┘
```

## 📁 Структура проекта

```
.
├── cmd/app              # Точка входа и инициализация зависимостей (Wire)
│   └── wire            # Настройка dependency injection
├── configs              # Конфигурация приложения (Viper)
├── internal
│   ├── delivery         # HTTP эндпоинты (Gin handlers)
│   ├── usecase          # Слой бизнес-логики
│   ├── domain           # Доменные модели и конвертации
│   └── repository       # Слой доступа к данным
│       └── postgres     # Реализация PostgreSQL
├── docs                 # Swagger-документация
├── envs                 # .env файлы
├── migrations           # Миграции базы данных
├── Makefile             # Команды сборки/запуска
└── docker-compose.yml   # Docker-инфраструктура
```

## 🚀 Быстрый старт

Убедитесь, что у вас установлен `Docker`, `Make`, `Go`, `Swag`.

### Запуск сервиса

```bash
make up
```

По умолчанию используется файл окружения: `./envs/.env.dev`.

### Остановка сервиса

```bash
make down
```

### Перезапуск

```bash
make restart
```

## 🛠 Разработка

### Сборка бинарника

```bash
make build
```

### Запуск приложения локально

```bash
make run
```

### Форматирование, валидация и зависимости

```bash
make fmt     # go fmt ./...
make vet     # go vet ./...
make tidy    # go mod tidy
```

### Запуск тестов

```bash
make test
```

### Регенерация Wire зависимостей

```bash
cd cmd/app/wire && go generate
```

## 📚 Swagger-документация

Сгенерировать:

```bash
make swag
```

После запуска доступно по адресу:

```
http://localhost:3000/swagger/index.html
```

## 📝 Пример Makefile команд

```bash
make up      # поднять контейнеры
make logs    # посмотреть логи
make swag    # сгенерировать Swagger
make build   # собрать Go-бинарник
```

## 🏗 Особенности шаблона

- **Чистая архитектура**: Разделение на слои delivery, usecase и repository
- **Dependency Injection**: Google Wire для compile-time DI
- **База данных**: PostgreSQL с пулом соединений
- **API**: REST API на фреймворке Gin
- **Graceful Shutdown**: Корректное завершение работы с таймаутом 5 секунд для HTTP сервера
- **Документация**: Swagger/OpenAPI документация
- **Конфигурация**: Viper для управления конфигурацией
- **Миграции**: Поддержка миграций базы данных

## 🔧 Конфигурация

Конфигурация управляется через переменные окружения или файлы `.env`. Значения по умолчанию заданы в `configs/config.go`.

Ключевые переменные окружения:
- `HTTP_PORT` - порт HTTP сервера (по умолчанию: 3000)
- `DATABASE_URL` - строка подключения к PostgreSQL
- `DB_MAX_CONNS` - максимальное количество соединений с БД (по умолчанию: 5)
- `DB_MAX_CONN_LIFETIME` - максимальное время жизни соединения (по умолчанию: 300s)

## 🔌 Dependency Injection (Wire)

Проект использует [Google Wire](https://github.com/google/wire) для dependency injection на этапе компиляции. Все компоненты приложения инициализируются через Wire провайдеры.

### Порядок инициализации

Wire автоматически разрешает и инициализирует зависимости в следующем порядке:

1. **Logger** (`ProvideLogger`) - создает структурированный логгер
2. **PostgreSQL Repository** (`ProvidePostgres`) - инициализирует пул соединений с БД
3. **Repository** (`ProvideRepository`) - создает обертку над репозиторием
4. **UseCase** (`ProvideUseCase`) - инициализирует слой бизнес-логики с репозиторием и логгером
5. **Handler** (`ProvideHandler`) - создает HTTP обработчики с usecase и логгером
6. **Engine** (`ProvideEngine`) - создает Gin роутер со всеми зарегистрированными маршрутами

### Структура Service

Структура `wire.Service` содержит все инициализированные компоненты:

```go
type Service struct {
    Postgres *postgres.Repository
    Logger   *slog.Logger
    Engine   *gin.Engine
}
```

### Добавление новых провайдеров

Чтобы добавить новые зависимости через Wire:

1. Создайте функцию-провайдер в `cmd/app/wire/providers.go`
2. Добавьте её в `wire.Build()` в `cmd/app/wire/wire.go`
3. Регенерируйте код Wire: `cd cmd/app/wire && go generate`

## 📦 Добавление новых функций

1. **Доменные модели**: Добавить в `internal/domain/`
2. **Бизнес-логика**: Реализовать в `internal/usecase/`
3. **HTTP-обработчики**: Добавить в `internal/delivery/`
4. **Операции с БД**: Реализовать в `internal/repository/postgres/`
5. **Маршруты**: Зарегистрировать в методе `RegisterRoutes()` вашего обработчика

---