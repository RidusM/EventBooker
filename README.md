# ebooker

Сервис бронирования мест на мероприятия с подтверждением по TTL. Принимает запросы через REST API, хранит данные в PostgreSQL, уведомляет пользователей через Email и Telegram-бота. Swagger UI доступен без дополнительной настройки.

## Содержание

- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Быстрый старт](#быстрый-старт)
- [Конфигурация](#конфигурация)
- [API](#api)
- [Схема БД](#схема-бд)
- [Разработка](#разработка)
- [Структура проекта](#структура-проекта)

## Возможности

- **REST API** — управление событиями, бронирование, подтверждение, управление пользователями
- **TTL-бронирование** — резерв места действует ограниченное время, по истечении — автоматически снимается
- **Двухэтапное бронирование** — `pending` → `confirmed`, отмена истёкших броней фоновым воркером
- **Email-уведомления** — отправка через SMTP при бронировании и подтверждении
- **Telegram-уведомления** — привязка аккаунта через one-time токен, уведомления через бота
- **Статистика событий** — количество забронированных, подтверждённых и свободных мест в реальном времени
- **Swagger UI** — `/swagger/index.html`

## Архитектура

```
HTTP-запрос (Gin)
       │
       ▼
 BookingHandler
       │
       ├── EventService      — создание и листинг событий, статистика
       ├── BookingService    — бронирование, подтверждение, отмена по TTL
       └── UserService       — регистрация, логин, привязка Telegram
               │
               ├── PostgreSQL (pgxpool)   — основное хранилище
               ├── SMTPNotifier           — email-уведомления
               └── TelegramNotifier       — уведомления через бота
```

Жизненный цикл бронирования:

```
pending ──(подтверждение до TTL)──► confirmed
        ──(TTL истёк)─────────────► cancelled  ← фоновый воркер
```

Статусы бронирования:

| Статус      | Описание                                      |
|-------------|-----------------------------------------------|
| `pending`   | Место зарезервировано, ожидает подтверждения  |
| `confirmed` | Бронирование подтверждено                     |
| `cancelled` | Отменено вручную или истёк TTL                |

## Быстрый старт

### Требования

- Docker и Docker Compose
- Go 1.21+ (только для локальной разработки)

### Запуск через Docker Compose (рекомендуется)

```bash
# 1. Клонировать репозиторий
git clone https://github.com/yourname/ebooker
cd ebooker

# 2. Скопировать и заполнить конфиг
cp configs/dev.env .env

# 3. Запустить всё (PostgreSQL + миграции + приложение)
docker compose up --build -d
```

Сервис будет доступен на http://localhost:8080.  
Swagger UI — http://localhost:8080/swagger/index.html.

### Запуск только инфраструктуры (для разработки)

```bash
# Поднять PostgreSQL
docker compose up db -d

# Запустить приложение
make run
```

## Конфигурация

Все параметры задаются через переменные окружения (файл `.env` или `configs/dev.env`).

### Приложение

| Переменная    | По умолчанию       | Описание                                    |
|---------------|--------------------|---------------------------------------------|
| `APP_NAME`    | `delayed-notifier` | Название сервиса                            |
| `APP_VERSION` | `1.0.0`            | Версия                                      |
| `ENV`         | `local`            | Окружение: `local`, `dev`, `staging`, `prod` |

### Сервис

| Переменная           | По умолчанию | Описание                                       |
|----------------------|--------------|------------------------------------------------|
| `SERVICE_BOOK_EVENT_TTL` | `15`     | TTL бронирования в минутах (мин: 3, макс: 1440) |

### База данных

| Переменная              | По умолчанию                                              | Описание                           |
|-------------------------|-----------------------------------------------------------|------------------------------------|
| `DB_DSN`                | `postgres://user:pass@localhost:5432/ebooker?sslmode=disable` | DSN подключения              |
| `DB_POOL_MAX`           | `20`                                                      | Максимальный размер пула соединений |
| `DB_CONN_ATTEMPTS`      | `5`                                                       | Число попыток подключения           |
| `DB_BASE_RETRY_DELAY`   | `100ms`                                                   | Базовая задержка между попытками   |
| `DB_MAX_RETRY_DELAY`    | `5s`                                                      | Максимальная задержка              |

### SMTP

| Переменная      | По умолчанию            | Описание              |
|-----------------|-------------------------|-----------------------|
| `SMTP_HOST`     | `smtp.gmail.com`        | SMTP-сервер           |
| `SMTP_PORT`     | `587`                   | Порт                  |
| `SMTP_USERNAME` | —                       | Логин                 |
| `SMTP_PASSWORD` | —                       | Пароль                |
| `SMTP_FROM`     | `noreply@example.com`   | Адрес отправителя     |

### Telegram

| Переменная  | Описание                        |
|-------------|---------------------------------|
| `TG_ALIAS`  | Username бота (без `@`)         |
| `TG_TOKEN`  | Токен бота из @BotFather        |

### HTTP-сервер

| Переменная                  | По умолчанию |
|-----------------------------|--------------|
| `HTTP_HOST`                 | `0.0.0.0`    |
| `HTTP_PORT`                 | `8080`       |
| `HTTP_READ_TIMEOUT`         | `5s`         |
| `HTTP_WRITE_TIMEOUT`        | `5s`         |
| `HTTP_IDLE_TIMEOUT`         | `60s`        |
| `HTTP_SHUTDOWN_TIMEOUT`     | `10s`        |
| `HTTP_READ_HEADER_TIMEOUT`  | `5s`         |
| `HTTP_MAX_HEADER_BYTES`     | `1048576`    |

### Logger

| Переменная          | По умолчанию                        |
|---------------------|-------------------------------------|
| `LOGGER_LEVEL`      | `info`                              |
| `LOGGER_FILENAME`   | `./logs/delayed-notifier.log`       |
| `LOGGER_MAX_SIZE`   | `100`                               |
| `LOGGER_MAX_BACKUPS`| `3`                                 |
| `LOGGER_MAX_AGE`    | `28`                                |

## API

Полная документация — http://localhost:8080/swagger/index.html

### События

#### `POST /events` — Создать событие

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Go Meetup #12",
    "description": "Митап сообщества Go-разработчиков",
    "date": "2026-07-01T18:00:00Z",
    "total_seats": 50,
    "booking_ttl_mins": 15
  }'
```

Ответ `201 Created`:

```json
{
  "id": "018f4c2a-1234-7abc-9def-000000000001",
  "title": "Go Meetup #12",
  "date": "2026-07-01T18:00:00Z",
  "total_seats": 50,
  "booking_ttl_mins": 15
}
```

#### `GET /events` — Список событий со статистикой

```bash
curl http://localhost:8080/events
```

#### `GET /events/{id}` — Детали события

```bash
curl http://localhost:8080/events/018f4c2a-1234-7abc-9def-000000000001
```

Ответ включает статистику: забронировано, подтверждено, свободных мест.

### Бронирование

#### `POST /events/{id}/book` — Забронировать место

```bash
curl -X POST http://localhost:8080/events/018f4c2a-1234-7abc-9def-000000000001/book \
  -H "Content-Type: application/json" \
  -d '{"user_id": "018f4c2a-0000-7abc-9def-000000000099"}'
```

Ответ `200 OK`:

```json
{
  "id": "018f4c2a-5678-7abc-9def-000000000002",
  "event_id": "018f4c2a-1234-7abc-9def-000000000001",
  "user_id": "018f4c2a-0000-7abc-9def-000000000099",
  "status": "pending",
  "expires_at": "2026-06-01T12:15:00Z"
}
```

#### `POST /bookings/confirm` — Подтвердить бронирование

Должен быть вызван до истечения `expires_at`.

```bash
curl -X POST http://localhost:8080/bookings/confirm \
  -H "Content-Type: application/json" \
  -d '{"booking_id": "018f4c2a-5678-7abc-9def-000000000002"}'
```

Ответ `201 Created`:

```json
{"message": "confirmed"}
```

### Пользователи и авторизация

#### `POST /auth/sign-up` — Регистрация

```bash
curl -X POST http://localhost:8080/auth/sign-up \
  -H "Content-Type: application/json" \
  -d '{"name": "Иван Иванов", "email": "ivan@example.com"}'
```

Ответ `201 Created`:

```json
{
  "user_id": "018f4c2a-0000-7abc-9def-000000000099",
  "message": "Registered via email"
}
```

#### `POST /auth/login` — Вход по email

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "ivan@example.com"}'
```

#### `GET /users/{id}` — Получить пользователя

```bash
curl http://localhost:8080/users/018f4c2a-0000-7abc-9def-000000000099
```

#### `GET /users` — Список пользователей

```bash
curl http://localhost:8080/users
```

#### `POST /users/{user_id}/link-token` — Привязать Telegram

Генерирует one-time токен (TTL 1 час). Пользователь переходит по ссылке и пишет боту `/start <token>`.

```bash
curl -X POST http://localhost:8080/users/018f4c2a-0000-7abc-9def-000000000099/link-token
```

Ответ `200 OK`:

```json
{
  "token": "a1b2c3d4e5f6",
  "link": "https://t.me/mybot?start=a1b2c3d4e5f6",
  "message": "Follow the link to connect Telegram",
  "expires_in": "1h"
}
```

### Система

#### `GET /health` — Проверка работоспособности

```bash
curl http://localhost:8080/health
# {"status":"ok","time":"2026-06-01T12:00:00Z"}
```

## Схема БД

```
events
  id UUID PK │ title │ description │ date │ total_seats │ booking_ttl_min

users
  id UUID PK │ name │ email UNIQUE │ telegram_id UNIQUE

bookings
  id UUID PK │ event_id FK → events │ user_id FK → users
  status: pending | confirmed | cancelled │ expires_at

user_link_tokens
  token PK │ user_id FK → users │ expires_at
```

Индексы:
- `idx_bookings_expiry` — по `(status, expires_at) WHERE status = 'pending'` — для воркера отмены
- `idx_bookings_event_status` — по `(event_id, status)` — для статистики событий
- `idx_users_email`, `idx_users_telegram_id` — частичные, только по непустым значениям

## Разработка

```bash
# Запустить приложение локально
make run

# Собрать бинарник
make build

# Запустить тесты
make test

# Линтер
make lint

# Форматирование
make format

# Сгенерировать Swagger
make swagger

# Поднять всё через Docker Compose
make compose-up

# Остановить
make compose-down
```

## Структура проекта

```
ebooker/
├── cmd/
│   └── ebooker/
│       └── main.go              # Точка входа
├── configs/
│   ├── dev.env
│   └── prod.env
├── docs/                        # Swagger (генерируется make swagger)
├── internal/
│   ├── app/
│   │   └── app.go               # Инициализация и запуск компонентов
│   ├── config/
│   │   └── config.go            # Конфигурация через env-переменные
│   ├── entity/                  # Доменные типы: Event, Booking, User
│   ├── service/                 # Бизнес-логика: EventService, BookingService, UserService
│   ├── repository/              # Работа с PostgreSQL
│   ├── notifier/
│   │   ├── smtp.go              # Email-уведомления
│   │   └── telegram.go          # Telegram-уведомления
│   └── transport/
│       └── http/                # Handlers, middleware, роутер (Gin)
├── migrations/                  # SQL-миграции (golang-migrate)
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── go.mod
```