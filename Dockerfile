FROM golang:1.23.4-alpine as build

RUN go version
RUN apk add git

COPY . /app
WORKDIR /app

RUN go mod download && go get -u ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o ./.bin/service ./cmd/service


# Используем минимальный базовый образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=0 /bin/app .

CMD ["./service"]
