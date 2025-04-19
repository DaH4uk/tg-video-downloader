FROM golang:1.24.2-alpine as build

# Убедимся, что Go установлен
RUN go version

# Устанавливаем git (необходим для go mod download)
RUN apk add --no-cache git

# Копируем исходный код в контейнер
COPY . /app
WORKDIR /app

# Скачиваем зависимости
RUN go mod download

# Прогоняем тесты
RUN go test -cover -v ./...

# Собираем бинарный файл
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/service ./cmd/service


# Используем минимальный базовый образ
FROM alpine:latest

# Определяем аргументы запуска
ARG TELEGRAM_BOT_TOKEN
ARG POSTGRES_DSN

# Устанавливаем их как переменные окружения
ENV TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN
ENV POSTGRES_DSN=$POSTGRES_DSN

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates

# Копируем бинарный файл из стадии сборки
COPY --from=build /app/service /root/service

# Устанавливаем рабочую директорию
WORKDIR /root

# Даем права на выполнение
RUN chmod +x /root/service

# Запускаем сервис
ENTRYPOINT ["/root/service"]
