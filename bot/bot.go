package bot

import (
	"strings"

	"fortio.org/log"
	"fortio.org/scli"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/repl"
)

var BotToken string

func checkNilErr(e error) {
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

func newMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {

	/* prevent bot responding to its own message
	this is achieved by looking into the message author id
	if message.author.id is same as bot.author.id then just return
	*/
	if message.Author.ID == discord.State.User.ID {
		return
	}

	// respond to user message if it contains `!help` or `!bye`
	switch {
	case strings.HasPrefix(message.Content, "!help"):
		_, err := discord.ChannelMessageSend(message.ChannelID, "Hello World😃")
		checkNilErr(err)
	case strings.HasPrefix(message.Content, "!bye"):
		_, err := discord.ChannelMessageSend(message.ChannelID, "Good Bye👋")
		checkNilErr(err)
		// add more cases if required
	case strings.HasPrefix(message.Content, "!grol"):
		res := repl.EvalString(message.Content[5:])
		_, err := discord.ChannelMessageSend(message.ChannelID, res)
		checkNilErr(err)
	}
}
