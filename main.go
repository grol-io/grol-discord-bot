package main

import (
	"flag"
	"math"
	"os"
	"runtime/debug"

	"fortio.org/log"
	"fortio.org/scli"
)

var (
	depth  = flag.Int("max-depth", 10000, "Maximum depth of recursion")
	maxLen = flag.Int("max-save-len", 1000, "Maximum len of saved identifiers, use 0 for unlimited")
)

func main() {
	num := flag.Int("n", 100, "Maximum number of messages to keep in memory for possible edit")
	scli.ServerMain()
	BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if BotToken == "" {
		log.Fatalf("DISCORD_BOT_TOKEN must be set")
	}
	AutoLoadSave = !(os.Getenv("GROL_DISABLE_AUTOSAVE") == "1")
	memlimit := debug.SetMemoryLimit(-1)
	if memlimit == math.MaxInt64 {
		log.Fatalf("Memory limit not set, please set GOMEMLIMIT=1GiB or similar")
	}
	Run(*num)
}
