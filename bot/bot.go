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
	discord, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Init discordgo.New error: %v", err)
	}

	// add a event handler
	discord.AddHandler(newMessage)

	// open session
	err = discord.Open()
	if err != nil {
		log.Fatalf("Init discordgo.Open error: %v", err)
	}
	defer discord.Close() // close session, after function termination

	// keep bot running until there is NO os interruption (ctrl + C)
	scli.UntilInterrupted()
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
	// Is this cached/efficient to keep doing?
	channel, err := session.State.Channel(message.ChannelID)
	var channelName string
	if err != nil {
		log.S(log.Error, "unable to get channel info", log.Any("err", err))
		channelName = "unknown"
	} else {
		channelName = channel.Name
	}
	server, err := session.State.Guild(channel.GuildID)
	var serverName string
	if err != nil {
		log.S(log.Error, "unable to get server info", log.Any("err", err))
		serverName = "unknown"
	} else {
		serverName = server.Name
	}
	log.S(log.Debug, "channel", log.Any("channel", channel))
	log.S(log.Info, "message",
		log.Any("from", message.Author.Username),
		log.Any("server", serverName),
		log.Any("channel", channelName),
		log.Any("content", message.Content))

	if !strings.HasPrefix(message.Content, "!grol") {
		return
	}

	res := repl.EvalString(message.Content[5:])
	log.S(log.Info, "response", log.String("response", res))
	_, err = session.ChannelMessageSend(message.ChannelID, "`"+res+"`")
	if err != nil {
		log.S(log.Error, "error", log.Any("err", err))
	}
}
