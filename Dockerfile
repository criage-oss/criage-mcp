# Многоэтапная сборка для минимизации размера образа
FROM golang:1.24.4-alpine AS builder

# Устанавливаем необходимые пакеты для сборки
RUN apk add --no-cache git ca-certificates

# Создаем рабочую директорию
WORKDIR /app

# Копируем go.mod для кеширования зависимостей (если есть)
COPY go.mod* ./

# Загружаем зависимости (если есть)
RUN if [ -f go.mod ]; then go mod download; fi

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.ServerVersion=1.0.0" -o criage-mcp-server .

# Финальный образ
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk add --no-cache ca-certificates tzdata

# Создаем пользователя для безопасности
RUN addgroup -g 1001 -S criage && \
    adduser -u 1001 -S criage -G criage

# Создаем рабочую директорию
WORKDIR /app

# Копируем исполняемый файл из стадии сборки
COPY --from=builder /app/criage-mcp-server /usr/local/bin/criage-mcp-server

# Копируем конфигурационные файлы
COPY config.example.json /app/config.json

# Создаем директории для данных
RUN mkdir -p /app/data && \
    chown -R criage:criage /app

# Переключаемся на непривилегированного пользователя
USER criage

# Переменные окружения
ENV CRIAGE_MCP_VERSION=1.0.0
ENV CRIAGE_MCP_CONFIG=/app/config.json

# Том для данных
VOLUME ["/app/data"]

# Точка входа
ENTRYPOINT ["criage-mcp-server"]

# Команда по умолчанию (MCP сервер работает через stdio)
CMD []

# Метаданные образа
LABEL maintainer="Criage Team"
LABEL version="1.0.0"
LABEL description="MCP сервер для интеграции Criage с AI инструментами"
LABEL org.opencontainers.image.source="https://github.com/criage-oss/criage-mcp"
LABEL org.opencontainers.image.documentation="https://criage.ru/mcp-server.html"
LABEL org.opencontainers.image.licenses="MIT"

# Информация о MCP протоколе
LABEL mcp.protocol.version="2024-11-05"
LABEL mcp.server.name="criage-mcp-server"
LABEL mcp.capabilities="tools,resources"
