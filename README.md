# URL Cutter

Простое Go-приложение на Gin. Добавлена интеграция с Sentry и готовый Docker-образ для деплоя на Render.

## Среда и переменные

- `PORT` — порт сервиса, по умолчанию `8080`.
- `DATABASE_URL` — строка подключения Postgres для миграций `goose`.
- `SENTRY_DSN` — DSN проекта Sentry. Если пусто, сбор ошибок отключена.

## Локальный запуск

```bash
go run .
```

Эндпоинты:
- `GET /ping` — проверка живости, ответ `pong`.
- `GET /debug-sentry` — генерирует тестовую ошибку в Sentry (если `SENTRY_DSN` задан), иначе сообщает, что Sentry выключен.

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

1) Создайте Web Service: Language — Docker, Instance Type — Free.  
2) Render возьмет `Dockerfile` из корня.  
3) Добавьте переменные окружения: `PORT=8080`, `DATABASE_URL`, `SENTRY_DSN`.  
4) Дождитесь сборки и убедитесь, что приложение доступно по HTTPS.  
5) Для проверки Sentry вызовите `/debug-sentry`.

## Прод-окружение

Ссылка на развернутое приложение: `<добавить URL Render после деплоя>`.

### Hexlet tests and linter status:
[![Actions Status](https://github.com/motokazmin/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/motokazmin/go-project-278/actions)