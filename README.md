# URL Cutter

Простое Go-приложение на Gin. Добавлена интеграция с Sentry и готовый Docker-образ для деплоя на Render.

## Среда и переменные

- `PORT` — порт сервиса, по умолчанию `8080`.
- `DATABASE_URL` — строка подключения Postgres для миграций `goose` и приложения (например, `postgres://user:pass@host:5432/db?sslmode=require`).
- `SENTRY_DSN` — DSN проекта Sentry. Если пусто, сбор ошибок отключена.
- `BASE_URL` — базовый адрес для формирования `short_url` (например, `https://example.com/r`).

## Локальный запуск

```bash
go run .
```

Эндпоинты:
- `GET /ping` — проверка живости, ответ `pong`.
- `GET /debug-sentry` — генерирует тестовую ошибку в Sentry (если `SENTRY_DSN` задан), иначе сообщает, что Sentry выключен.
- CRUD API:
  - `GET /api/links` — список ссылок.
  - `POST /api/links` — создать ссылку (`original_url` обязателен, `short_name` необязателен).
  - `GET /api/links/:id` — получить по id.
  - `PUT /api/links/:id` — обновить `original_url`/`short_name`.
  - `DELETE /api/links/:id` — удалить.  
  Коды: `404 Not Found` на отсутствующий id, `409 Conflict` при конфликте `short_name`.

## Сборка и запуск в Docker

```bash
docker build -t urlcutter .
docker run --rm -p 8080:8080 \
  -e PORT=8080 \
  -e DATABASE_URL="postgres://..." \
  -e SENTRY_DSN="https://..." \
  urlcutter
```

Контейнер стартует через `bin/run.sh`, который перед запуском приложения применяет миграции `goose`.

## Деплой на Render

1) Создаем Web Service: Language — Docker, Instance Type — Free.  
2) Render возьмет `Dockerfile` из корня.  
3) Добавим переменные окружения: `PORT=8080`, `DATABASE_URL`, `SENTRY_DSN`.  
4) Для проверки Sentry вызвать `/debug-sentry`.

## Миграции и sqlc

- Миграции (goose) лежат в `db/migrations`. Запуск в контейнере выполняется через `bin/run.sh`.
- Конфиг `sqlc.yaml`, запросы в `db/queries`, сгенерированный код в `internal/db`.

### Hexlet tests and linter status:
[![Actions Status](https://github.com/motokazmin/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/motokazmin/go-project-278/actions)


docker run --rm -p 8080:8080 \
  --add-host=host.docker.internal:host-gateway \
  -e PORT=8080 \
  -e BASE_URL="http://localhost:8080" \
  -e DATABASE_URL="postgres://user:pass@host.docker.internal:5432/db?sslmode=disable" \
  -e SENTRY_DSN="..." \
  urlcutter

  npm start поднимает одновременно API (8080) и фронт (5173). CORS разрешён для FRONTEND_ORIGIN (по умолчанию http://localhost:5173).