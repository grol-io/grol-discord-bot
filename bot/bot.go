package bot

import (
	"strings"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/scli"
	"fortio.org/version"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/repl"
)

var BotToken string

func Run() {
	// create a session
	session, err := discordgo.New("Bot " + BotToken)
	session.StateEnabled = true
	if err != nil {
		log.Fatalf("Init discordgo.New error: %v", err)
	}

	// add a event handler
	session.AddHandler(newMessage)

	// open session
	err = session.Open()
	if err != nil {
		log.Fatalf("Init discordgo.Open error: %v", err)
	}
	defer session.Close() // close session, after function termination

	// keep bot running until there is NO os interruption (ctrl + C)
	scli.UntilInterrupted()
}

func handleDM(session *discordgo.Session, message *discordgo.MessageCreate) {
	log.S(log.Info, "direct-message",
		log.Any("from", message.Author.Username),
		log.Any("content", message.Content))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	what := message.Content
	if strings.HasPrefix(message.Content, "!grol") {
		what = what[5:]
	}
	evalAndReply(session, "dm-reply", message.ChannelID, what)
}

var growlVersion, _, _ = version.FromBuildInfoPath("grol.io/grol")

func removeTripleBackticks(s string) string {
	s = strings.TrimPrefix(s, "```grol")
	s = strings.TrimPrefix(s, "```go")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return s
}

func evalAndReply(session *discordgo.Session, info, channelID, input string) {
	var res string
	input = strings.TrimSpace(input)
	switch input {
	case "":
		fallthrough
	case "info":
		fallthrough
	case "help":
		res = "Grol bot help: grol bot evaluates grol language fragments, as simple as expressions like `1+1`" +
			" and as complex as defining closures, using map, arrays, etc... the syntax is similar to go.\n\n" +
			"also supported `!grol version`, `!grol source`, `!grol buildinfo`"
	case "source":
		res = "[github.com/grol-io/grol-discord-bot](<https://github.com/grol-io/grol-discord-bot>)" +
			" and [grol-io](<https://grol.io>)"
	case "version":
		res = "Grol bot version: " + cli.ShortVersion + ", `grol` language version " + growlVersion + ")"
	case "buildinfo":
		res = "```" + cli.FullVersion + "```"
	default:
		// TODO: stdout vs stderr vs result. https://github.com/grol-io/grol/issues/33
		// TODO: Maybe better quoting.
		input = removeTripleBackticks(input)
		var errs []string
		res, errs = repl.EvalString(input)
		if len(errs) > 0 {
			res = "```diff"
			for _, e := range errs {
				res += "\n- " + e
			}
			res += "\n```"
		} else {
			res = "```go\n" + res + "\n```"
		}
	}
	log.S(log.Info, info, log.String("response", res))
	reply(session, channelID, res)
}

func reply(session *discordgo.Session, channelID, response string) {
	_, err := session.ChannelMessageSend(channelID, response)
	if err != nil {
		log.S(log.Error, "error", log.Any("err", err))
	}
}

func newMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	/* prevent bot responding to its own message
	this is achieved by looking into the message author id
	if message.author.id is same as bot.author.id then just return
	*/
	log.S(log.Debug, "message", log.Any("message", message))
	if message.Author.ID == session.State.User.ID {
		return
	}
	isDM := message.GuildID == ""
	if isDM {
		handleDM(session, message)
		return
	}
	// Is this cached/efficient to keep doing?
	channel, err := session.State.Channel(message.ChannelID)
	var channelName string
	if err != nil {
		log.S(log.Error, "unable to get channel info", log.Any("err", err))
		channelName = "unknown"
	} else {
		channelName = channel.Name
	}
	server, err := session.State.Guild(message.GuildID)
	var serverName string
	if err != nil {
		log.S(log.Error, "unable to get server info", log.Any("err", err))
		serverName = "unknown"
	} else {
		serverName = server.Name
	}
	if !strings.HasPrefix(message.Content, "!grol") {
		return
	}
	log.S(log.Info, "channel-message",
		log.Any("from", message.Author.Username),
		log.Any("server", serverName),
		log.Any("channel", channelName),
		log.Any("content", message.Content))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	evalAndReply(session, "channel-response", message.ChannelID, message.Content[5:])
}
