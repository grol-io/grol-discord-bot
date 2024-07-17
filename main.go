package main

import (
	"flag"
	"os"

	"fortio.org/log"
	"fortio.org/scli"
)

func main() {
	num := flag.Int("n", 100, "Maximum number of messages to keep in memory for possible edit")
	scli.ServerMain()
	BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if BotToken == "" {
		log.Fatalf("DISCORD_BOT_TOKEN must be set")
	}
	Run(*num)
}
