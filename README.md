# A custom WebHook server for Grafana which will send message through a Proxy Server

Support only for Discord and Telegram

Remember to add these environment variables into your .env:
```
# Telegram
BOT_TOKEN=<YOUR_TELEGRAM_BOT_TOKEN>
CHAT_ID=<YOUR_TELEGRAM_CHAT_ID>
TELEGRAM_DISABLED=true                                              # Set something different with true to enabled it

# Proxy
PROXY_URL=<YOUR_PROXY_URL>                                          # For example: http://1.2.3.4:1234
PROXY_TYPE=<YOUR_PROXY_TYPE>                                        # Just support for socks5
PROXY_USER=<YOUR_PROXY_USERNAME>
PROXY_PASS=<YOUR_PROXY_PASSWORD>

# Discord
DISCORD_URL=<YOUR_DISCORD_WEBHOOK_URL>
DISCORD_BOT_TOKEN=<YOUR_DISCORD_BOT_TOKEN>
DISCORD_APPLICATION_ID=<YOUR_DISCORD_APPLICATION_ID>
DISCORD_PUBLIC_KEY=<YOUR_DISCORD_PUBLIC_KEY>
DISCORD_CHANNEL_ID=<YOUR_DISCORD_CHANNEL_ID>

# Mongodb
MONGODB_URI=mongodb://mongodb:27017
MONGODB_DATABASE=grafana-alerts
```

## I. Instruction for run binaries file

> If you run binaries file, remmeber to change MONGODB_URI to your mongodb uri

```bash
go build -ldflags="-s -w"
```

### Run
```bash
./webhook-server
```

## II. Instruction for run docker compose

```bash
docker-compose up -d
```

## II. Results Demo

### 1. Telegram
![TELEGRAM NOTIFY](screenshot/telegram-notify.png)

### 2. Discord
![DISCORD NOTIFY](screenshot/discord-notify.png)
