# A custom WebHook server for Grafana which will send message through a Proxy Server

Support only for Discord and Telegram

## I. Instruction for run binaries file
```bash
go build -ldflags="-s -w"
```

### Run
```bash
.\webhook-proxy
```

## II. Instruction for run docker container

### 1. Build Docker image
```bash
docker build -t webhook-proxy .
```

### 2.
```bash
docker run -dp 8080:8080 --name webhook-proxy webhook-proxy

```

## II. Results Demo

### 1. Telegram
![TELEGRAM NOTIFY](screenshot/telegram-notify.png)

### 2. Discord
![DISCORD NOTIFY](screenshot/discord-notify.png)
