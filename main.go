package main

import (
	"flag"
	"os"

	"fortio.org/log"
	"fortio.org/scli"
	"grol.io/grol-discord-bot/bot"
)

func main() {
	num := flag.Int("n", 100, "Maximum number of messages to keep in memory for possible edit")
	scli.ServerMain()
	bot.BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if bot.BotToken == "" {
		log.Fatalf("DISCORD_BOT_TOKEN must be set")
	}
	bot.Run(*num)
}
