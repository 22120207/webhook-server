# A custom WebHook server for Grafana which will send message through a Proxy Server

Support only for Discord and Telegram

## I. Installation
```bash
$ go build -ldflags="-s -w"
```

### Run
```bash
.\grafana-webhook-proxy
```

## II. Results Demo
#### Telegram
![TELEGRAM NOTIFY](screenshot/telegram-notify.png)

#### Discord
![DISCORD NOTIFY](screenshot/discord-notify.png)