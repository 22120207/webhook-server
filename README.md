# A custom WebHook server for Grafana which will send message through a Proxy Server

Support only for Discord and Telegram

## I. Installation
```bash
$ go build -ldflags="-s -w"
```

### Run
```bash
.\webhook-proxy
```

## II. Results Demo
#### 1. Telegram
![TELEGRAM NOTIFY](screenshot/telegram-notify.png)

#### 2. Discord
![DISCORD NOTIFY](screenshot/discord-notify.png)
