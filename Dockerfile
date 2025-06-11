FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server .

FROM debian:bullseye-slim

COPY --from=builder /app/server /usr/local/bin/server

EXPOSE 8080

ENTRYPOINT ["server"]
