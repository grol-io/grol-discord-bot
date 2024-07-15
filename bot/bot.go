package bot

import (
	"strings"

	"fortio.org/log"
	"fortio.org/scli"
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
	evalAndReply(session, "dm-reply", message.ChannelID, message.Content)
}

func evalAndReply(session *discordgo.Session, info, channelID, input string) {
	// TODO: stdout vs stderr vs result.
	res := repl.EvalString(input)
	log.S(log.Info, info, log.String("response", res))
	reply(session, channelID, res)
}

func reply(session *discordgo.Session, channelID, response string) {
	// TODO: Maybe better quoting.
	_, err := session.ChannelMessageSend(channelID, "`"+response+"`")
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
	log.S(log.Debug, "channel", log.Any("channel", channel))
	log.S(log.Info, "channel-message",
		log.Any("from", message.Author.Username),
		log.Any("server", serverName),
		log.Any("channel", channelName),
		log.Any("content", message.Content))

	if !strings.HasPrefix(message.Content, "!grol") {
		return
	}
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	evalAndReply(session, "channel-response", message.ChannelID, message.Content[5:])
}
