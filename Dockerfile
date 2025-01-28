FROM golang:1.23.4-alpine as build

# Убедимся, что Go установлен
RUN go version

# Устанавливаем git (необходим для go mod download)
RUN apk add --no-cache git

# Копируем исходный код в контейнер
COPY . /app
WORKDIR /app

# Скачиваем зависимости
RUN go mod download

# Собираем бинарный файл
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/service ./cmd/service


# Используем минимальный базовый образ
FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates

# Копируем бинарный файл из стадии сборки
COPY --from=build /app/service /usr/local/bin/service

# Устанавливаем рабочую директорию
WORKDIR /root/

# Даем права на выполнение
RUN chmod +x /usr/local/bin/service

# Запускаем сервис
CMD ["service"]
