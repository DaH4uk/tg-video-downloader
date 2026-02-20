FROM golang:1.26-alpine as build

# Убедимся, что Go установлен
RUN go version

# Устанавливаем git (необходим для go mod download)
RUN apk add --no-cache git

# Копируем исходный код в контейнер
COPY . /app
WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org
# Скачиваем зависимости
RUN for i in 1 2 3 4 5; do go mod download && break || sleep 3; done

## Прогоняем тесты
#RUN go test -cover -v ./...

RUN ls

# Собираем бинарный файл
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/service ./cmd/service


# Используем минимальный базовый образ
FROM alpine:latest

# Определяем аргументы запуска
ARG TELEGRAM_BOT_TOKEN

# Устанавливаем их как переменные окружения
ENV TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates

# Копируем .env файл из стадии сборки
COPY --from=build /app/.env /root/.env

# Копируем бинарный файл из стадии сборки
COPY --from=build /app/service /root/service

# Устанавливаем рабочую директорию
WORKDIR /root

# Даем права на выполнение
RUN chmod +x /root/service

# Запускаем сервис
ENTRYPOINT ["/root/service"]
