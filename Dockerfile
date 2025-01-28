# Используем минимальный базовый образ
FROM alpine:latest

# Копируем собранный бинарник
COPY ./bin/service /app/service

# Делаем бинарник исполняемым
RUN chmod +x /app/service

# Указываем рабочую директорию
WORKDIR /app

# Команда запуска контейнера
CMD ["/app/service"]
