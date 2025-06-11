# syntax=docker/dockerfile:1

# Stage 1: Build Go binary and get certificates
FROM golang:1.23-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-proxy

# Stage 2: Final minimal image using scratch
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/webhook-proxy /webhook-proxy
COPY --from=builder /app/.env /app/.env

# Expose port and run
EXPOSE 8080
ENTRYPOINT ["/webhook-proxy"]