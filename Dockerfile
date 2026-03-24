FROM --platform=$BUILDPLATFORM golang:1.26-alpine as build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Копируем исходный код вместе с вендорными зависимостями
COPY . .

## Прогоняем тесты
#RUN go test -cover -v ./...

# Собираем бинарный файл под целевую платформу
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -mod=vendor -o /app/service ./cmd/service


# Используем минимальный базовый образ
FROM alpine:latest

# Определяем аргументы запуска
ARG TELEGRAM_BOT_TOKEN

# Устанавливаем их как переменные окружения
ENV TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN
ENV HTTP_PROXY=$HTTP_PROXY

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
