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

var msgSet *FixedMap[string, string]

func Run(maxHistoryLength int) {
	msgSet = NewFixedMap[string, string](maxHistoryLength)
	// create a session
	session, err := discordgo.New("Bot " + BotToken)
	session.StateEnabled = true
	if err != nil {
		log.Fatalf("Init discordgo.New error: %v", err)
	}

	// add event handlers
	session.AddHandler(newMessage)
	session.AddHandler(updateMessage)

	// open session
	err = session.Open()
	if err != nil {
		log.Fatalf("Init discordgo.Open error: %v", err)
	}
	defer session.Close() // close session, after function termination

	// keep bot running until there is NO os interruption (ctrl + C)
	scli.UntilInterrupted()
}

func handleDM(session *discordgo.Session, message *discordgo.Message, replyID string) {
	log.S(log.Info, "direct-message",
		log.Any("from", message.Author.Username),
		log.Any("content", message.Content))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	what := strings.TrimPrefix(message.Content, "!grol")
	replyID = evalAndReply(session, "dm-reply", message.ChannelID, what, replyID)
	msgSet.Add(message.ID, replyID)
}

var growlVersion, _, _ = version.FromBuildInfoPath("grol.io/grol")

func removeTripleBackticks(s string) string {
	s = strings.TrimPrefix(s, "```grol")
	s = strings.TrimPrefix(s, "```go")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return s
}

// returns the id of the reply.
func evalAndReply(session *discordgo.Session, info, channelID, input string, replyID string) string {
	var res string
	input = strings.TrimSpace(input) // we do it again so "   !grol    help" works
	switch input {
	case "":
		fallthrough
	case "info":
		fallthrough
	case "help":
		res = "💡 Grol bot help: grol bot evaluates grol language fragments, as simple as expressions like `1+1`" +
			" and as complex as defining closures, using map, arrays, etc... the syntax is similar to go (without :=).\n\n" +
			"also supported `!grol version`, `!grol source`, `!grol buildinfo`"
	case "source":
		res = "📄 [github.com/grol-io/grol-discord-bot](<https://github.com/grol-io/grol-discord-bot>)" +
			" and [grol-io](<https://grol.io>)"
	case "version":
		res = "📦 Grol bot version: " + cli.ShortVersion + ", `grol` language version " + growlVersion
	case "buildinfo":
		res = "📦ℹ️```" + cli.FullVersion + "```"
	case "bug":
		res = "🐞 Please report any issue or suggestion at [github.com/grol-io/grol-discord-bot/issues](<https://github.com/grol-io/grol-discord-bot/issues>)"
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
	return reply(session, channelID, res, replyID)
}

func reply(session *discordgo.Session, channelID, response, replyID string) string {
	var err error
	if replyID != "" {
		_, err = session.ChannelMessageEdit(channelID, replyID, response)
	} else {
		var reply *discordgo.Message
		reply, err = session.ChannelMessageSend(channelID, response)
		replyID = reply.ID
	}
	if err != nil {
		log.S(log.Error, "error", log.Any("err", err))
	}
	return replyID
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
	handleMessage(session, message, "")
}

func handleMessage(session *discordgo.Session, message *discordgo.MessageCreate, replyID string) {
	isDM := message.GuildID == ""
	message.Content = strings.TrimSpace(message.Content)
	if isDM {
		handleDM(session, message.Message, replyID)
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
	replyID = evalAndReply(session, "channel-response", message.ChannelID, message.Content[5:], replyID)
	msgSet.Add(message.ID, replyID)
}

func updateMessage(session *discordgo.Session, message *discordgo.MessageUpdate) {
	log.S(log.Debug, "message update", log.Any("message", message))
	if message.Author.ID == session.State.User.ID {
		return
	}
	reply, found := msgSet.Get(message.ID)
	if !found {
		log.S(log.Debug, "message not handled before", log.Any("id", message.ID))
		return
	}
	log.S(log.Info, "message edit detected", log.Any("id", message.ID), log.Any("reply", reply), log.String("new-content", message.Content))
	handleMessage(session, &discordgo.MessageCreate{Message: message.Message}, reply)
}
