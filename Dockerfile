FROM golang:1.23.4-alpine

WORKDIR /app
COPY ./bin /app

CMD ["/app/bin/service"]
