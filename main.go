package main

func main() {
	server := RestController{
		Telegram: &TelegramSender{},
		Discord:  &DiscordSender{},
	}

	server.Start()
}
