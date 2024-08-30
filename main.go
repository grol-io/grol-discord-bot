package main

import (
	_ "embed"
	"flag"
	"math"
	"os"
	"runtime/debug"
	"time"

	"fortio.org/log"
	"fortio.org/scli"
)

var (
	//go:embed discord.gr
	libraryCode string
	depth       = flag.Int("max-depth", 2500, "Maximum depth of recursion")
	maxLen      = flag.Int("max-save-len", 2000, "Maximum len of saved identifiers, use 0 for unlimited")
	maxDur      = flag.Duration("max-duration", 3*time.Second, "Maximum duration of scripts, use 0 for unlimited")
	panicF      = flag.Bool("panic", false, "Don't catch panic (DEV only)")
)

func main() {
	num := flag.Int("n", 100, "Maximum number of messages to keep in memory for possible edit")
	scli.ServerMain()
	BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if BotToken == "" {
		log.Fatalf("DISCORD_BOT_TOKEN must be set")
	}
	AutoSave = !(os.Getenv("GROL_DISABLE_AUTOSAVE") == "1")
	memlimit := debug.SetMemoryLimit(-1)
	if memlimit == math.MaxInt64 {
		log.Fatalf("Memory limit not set, please set GOMEMLIMIT=1GiB or similar")
	}
	BotAdmin = os.Getenv("DISCORD_BOT_ADMIN")
	Run(*num)
}
