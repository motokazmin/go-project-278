### 1) Build frontend
FROM node:24-alpine AS frontend-builder
WORKDIR /build/frontend

COPY package*.json ./
# Удалено кэширование: npm-зависимости будут скачиваться при каждой сборке
RUN npm ci --prefer-offline --no-audit

### 2) Build backend
FROM golang:1.25-alpine AS backend-builder
ENV GOPROXY=direct
ENV GOFLAGS=-mod=vendor

WORKDIR /build/code

COPY . .

# Удалено кэширование: Go-модули будут собираться при каждой сборке
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install github.com/pressly/goose/v3/cmd/goose && \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/app .

### 3) Runtime (Финальный образ)
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata bash caddy curl

WORKDIR /app

COPY --from=backend-builder /build/app /app/bin/app
COPY --from=frontend-builder /build/frontend/node_modules/@hexlet/project-url-shortener-frontend/dist /app/public

ENV PORT=8080 \
    BASE_URL=http://localhost:8080 \
    FRONTEND_ORIGIN=http://localhost:5173

COPY --from=backend-builder /build/code/db/migrations /app/db/migrations
COPY --from=backend-builder /go/bin/goose /usr/local/bin/goose

COPY bin/run.sh /app/bin/run.sh
RUN chmod +x /app/bin/run.sh

COPY Caddyfile /etc/caddy/Caddyfile

EXPOSE 80

CMD ["/app/bin/run.sh"]