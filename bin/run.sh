#!/usr/bin/env bash

set -euo pipefail

# 1. Диагностика: Вывод всех переменных окружения
echo "--- ENVIRONMENT VARIABLES ---"
env
echo "-----------------------------"

# 2. Вывод целевой переменной
echo "DATABASE_URL is set to: ${DATABASE_URL}"


echo "[run.sh] Starting service"

echo "[run.sh] Running DB migrations"
goose -dir ./db/migrations postgres "${DATABASE_URL}" up

echo "[run.sh] Starting Caddy"
caddy run --config /etc/caddy/Caddyfile &

echo "[run.sh] Starting Go app"
exec /app/bin/app

