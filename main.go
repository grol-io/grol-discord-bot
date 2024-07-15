package main

import (
	"os"

	"fortio.org/log"
	"fortio.org/scli"
	"grol.io/grol-discord-bot/bot"
)

func main() {
	scli.ServerMain()
	bot.BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if bot.BotToken == "" {
		log.Fatalf("DISCORD_BOT_TOKEN must be set")
	}
	bot.Run() // call the run function of bot/bot.go
}
