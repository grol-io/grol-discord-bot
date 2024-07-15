package bot

import (
	"strings"

	"fortio.org/log"
	"fortio.org/scli"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/repl"
)

var BotToken string

func checkNilErr(e error) { // TODO: replace by less brutal error handling
	if e != nil {
		log.Fatalf("Error: %v", e)
	}
}

func Run() {

	// create a session
	discord, err := discordgo.New("Bot " + BotToken)
	checkNilErr(err)

	// add a event handler
	discord.AddHandler(newMessage)

	// open session
	err = discord.Open()
	checkNilErr(err)
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
	channel, err := session.State.Channel(message.ChannelID)
	checkNilErr(err)
	server, err := session.State.Guild(channel.GuildID)
	checkNilErr(err)
	log.S(log.Debug, "channel", log.Any("channel", channel))
	log.S(log.Info, "message",
		log.Any("from", message.Author.Username),
		log.Any("server", server.Name),
		log.Any("channel", channel.Name),
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
