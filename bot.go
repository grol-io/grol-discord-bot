package main

import (
	"strconv"
	"strings"
	"time"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/scli"
	"fortio.org/version"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol-discord-bot/fixedmap"
	"grol.io/grol/extensions"
	"grol.io/grol/repl"
)

var (
	BotToken     string
	AutoLoadSave bool
	// State for edit to replies.
	msgSet       *fixedmap.FixedMap[string, string]
	botStartTime time.Time
)

const Unknown = "unknown"

func Run(maxHistoryLength int) {
	botStartTime = time.Now()
	extCfg := extensions.Config{
		HasLoad:           true,
		HasSave:           true,
		LoadSaveEmptyOnly: true,
	}
	err := extensions.Init(&extCfg)
	if err != nil {
		log.Fatalf("Grol extensions init error: %v", err)
	}
	msgSet = fixedmap.NewFixedMap[string, string](maxHistoryLength)
	// create a session
	session, err := discordgo.New("Bot " + BotToken)
	session.StateEnabled = true
	if err != nil {
		log.Fatalf("Init discordgo.New error: %v", err)
	}

	// add event handlers
	session.AddHandler(newMessage)
	session.AddHandler(updateMessage)
	session.AddHandler(interactionCreate)
	session.AddHandler(deleteMessage)

	// open session
	err = session.Open()
	if err != nil {
		log.Fatalf("Init discordgo.Open error: %v", err)
	}
	defer session.Close() // close session, after function termination

	registerCommands(session)
	log.Infof("Bot is now running with AutoLoadSave=%t.  Press CTRL-C or SIGTERM to exit.", AutoLoadSave)
	// keep bot running until there is NO os interruption (ctrl + C)
	scli.UntilInterrupted()
}

func updateMap(msgID, replyID string) {
	node, isNew := msgSet.Add(msgID, replyID)
	msg := "Updated message in history"
	if isNew {
		msg = "Added new message to history"
	}
	log.S(log.Verbose, msg, log.Any("msgID", msgID), log.Any("replyID", replyID))
	if node != nil {
		log.S(log.Verbose, "Evicted message from history", log.Any("msgID", node.Key), log.Any("replyID", node.Value))
	}
}

func handleDM(session *discordgo.Session, message *discordgo.Message, replyID string) {
	log.S(log.Info, "direct-message",
		log.Any("from", message.Author.Username),
		log.Any("content", message.Content))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	formatMode := strings.HasPrefix(message.Content, formatModeStr)
	compactMode := strings.HasPrefix(message.Content, compactModeStr)
	var what string
	switch {
	case formatMode:
		what = strings.TrimPrefix(message.Content, formatModeStr)
	case compactMode:
		what = strings.TrimPrefix(message.Content, compactModeStr)
	default:
		what = strings.TrimPrefix(message.Content, grolPrefix)
	}
	replyID = evalAndReply(session, "dm-reply", message.ChannelID, what, replyID, formatMode, compactMode)
	updateMap(message.ID, replyID)
}

var (
	growlVersion, _, _ = version.FromBuildInfoPath("grol.io/grol")
	grolPrefix         = "!grol"
	formatModeStr      = grolPrefix + " -f"
	compactModeStr     = grolPrefix + " -c"
)

func RemoveTripleBackticks(s string) string {
	// Extract the code in between triple backticks, ignoring the language tag if any.
	buf := strings.Builder{}
	first := true
	needNewline := false
	for {
		i := strings.Index(s, "```")
		if i == -1 {
			if first {
				buf.WriteString(s)
			}
			break
		}
		if needNewline {
			buf.WriteString("\n") // separate from previous set that didn't end with a newline.
		}
		first = false
		s = s[i:]
		s = strings.TrimPrefix(s, "```grol")
		s = strings.TrimPrefix(s, "```go")
		s = strings.TrimPrefix(s, "```")
		j := strings.Index(s, "```")
		if j == -1 {
			buf.WriteString(s)
			break
		}
		needNewline = (s[j-1] != '\n')
		buf.WriteString(s[:j])
		s = s[j+3:]
	}
	return strings.TrimSpace(buf.String())
}

func UptimeString(startTime time.Time) string {
	return DurationString(time.Since(startTime))
}

// DurationString returns a human readable string for a duration.
// Expressed in days, hours, minutes, seconds and 10th of second.
// days, hours etc are omitted if they are 0.
func DurationString(d time.Duration) string {
	rounded := d.Round(100 * time.Millisecond)
	// get number of days out:
	oneDay := 24 * time.Hour
	days := int(rounded / oneDay)
	if days == 0 {
		return rounded.String()
	}
	rounded -= time.Duration(days) * oneDay
	return strconv.Itoa(days) + "d" + rounded.String()
}

func eval(input string, formatMode bool, compactMode bool) string {
	var res string
	input = strings.TrimSpace(input) // we do it again so "   !grol    help" works
	switch input {
	case "":
		fallthrough
	case "help":
		res = "💡 Grol bot help: grol bot evaluates grol language fragments, as simple as expressions like `1+1`" +
			" and as complex as defining closures, using map, arrays, etc... the syntax is similar to go (without needing " +
			"`:=`, plain `=` is enough). Use `info` to see all functions, keywords, etc...\n\n" +
			"Either in DM or with `!grol` prefix (or `!grol -f` for also showing formatted code, `-c` in compact mode) in a channel, " +
			"you can type any grol code and the bot will evaluate it (only code blocks if there are any).\n\n" +
			"Also supported `!grol version`, `!grol source`, `!grol buildinfo`, `!grol bug`.\n\n" +
			"You can also try the /grol command, answers will be visible only to you!"
	case "source":
		res = "📄 [github.com/grol-io/grol-discord-bot](<https://github.com/grol-io/grol-discord-bot>)" +
			" and [grol-io](<https://grol.io>)"
	case "uptime":
		fallthrough
	case "version":
		res = "📦 Grol bot version: " + cli.ShortVersion + ", `grol` language version " + growlVersion +
			" ⏰ Uptime: " + UptimeString(botStartTime)
	case "buildinfo":
		res = "📦ℹ️```" + cli.FullVersion + "```"
	case "bug":
		res = "🐞 Please report any issue or suggestion at " +
			"[github.com/grol-io/grol-discord-bot/issues](<https://github.com/grol-io/grol-discord-bot/issues>)"
	default:
		// TODO: stdout vs stderr vs result. https://github.com/grol-io/grol/issues/33
		//   !grol
		//   ```go
		//   1+1
		//   ```
		//   look at the result of 1+1
		// in a single message and not get errors on the extra text (meanwhile, add //).
		input = RemoveTripleBackticks(input)
		cfg := repl.Options{
			All:      true,
			ShowEval: true,
			NoColor:  true,
			Compact:  compactMode,
			AutoLoad: AutoLoadSave,
			AutoSave: AutoLoadSave,
		}
		evalres, errs, formatted := repl.EvalStringWithOption(cfg, input)
		if formatMode || compactMode {
			res = formatModeStr
			if compactMode {
				res = compactModeStr
			}
			res += "\n```go\n" + formatted + "``` produces: "
		}
		evalres = strings.TrimSpace(evalres)
		if evalres != "" {
			res += "```go\n" + evalres + "\n```\n"
		} else {
			res += "no output.\n"
		}
		if len(errs) > 0 {
			res += "```diff"
			for _, e := range errs {
				res += "\n- " + e
			}
			res += "\n```"
		}
	}
	return res
}

// returns the id of the reply.
func evalAndReply(session *discordgo.Session, info, channelID, input string,
	replyID string, formatMode, compactMode bool,
) string {
	res := eval(input, formatMode, compactMode)
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
		channelName = Unknown
	} else {
		channelName = channel.Name
	}
	server, err := session.State.Guild(message.GuildID)
	var serverName string
	if err != nil {
		log.S(log.Error, "unable to get server info", log.Any("err", err))
		serverName = Unknown
	} else {
		serverName = server.Name
	}
	if !strings.HasPrefix(message.Content, grolPrefix) {
		if replyID != "" {
			// delete the reply if it's not a grol command anymore
			log.S(log.Info, "no prefix anymore, deleting previous reply", log.Any("replyID", replyID))
			err := session.ChannelMessageDelete(message.ChannelID, replyID)
			if err != nil {
				log.S(log.Error, "unable to delete message", log.Any("err", err))
			}
		}
		return
	}
	formatMode := strings.HasPrefix(message.Content, formatModeStr)
	compactMode := strings.HasPrefix(message.Content, compactModeStr)
	var content string
	switch {
	case formatMode:
		content = message.Content[len(formatModeStr):]
	case compactMode:
		content = message.Content[len(compactModeStr):]
	default:
		content = message.Content[len(grolPrefix):]
	}
	log.S(log.Info, "channel-message",
		log.Any("from", message.Author.Username),
		log.Any("server", serverName),
		log.Any("channel", channelName),
		log.Any("content", message.Content),
		log.Bool("format", formatMode))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	replyID = evalAndReply(session, "channel-response", message.ChannelID, content, replyID, formatMode, compactMode)
	updateMap(message.ID, replyID)
}

func updateMessage(session *discordgo.Session, message *discordgo.MessageUpdate) {
	log.S(log.Debug, "message update", log.Any("message", message))
	if message.Author.ID == session.State.User.ID { // self update bail?
		return
	}
	reply, found := msgSet.Get(message.ID)
	if !found {
		log.S(log.Debug, "message not handled before", log.Any("id", message.ID))
		return
	}
	log.S(log.Info, "message edit detected",
		log.Any("id", message.ID),
		log.Any("reply", reply),
		log.String("new-content", message.Content))
	handleMessage(session, &discordgo.MessageCreate{Message: message.Message}, reply)
}

func deleteMessage(session *discordgo.Session, message *discordgo.MessageDelete) {
	log.S(log.Debug, "message delete", log.Any("message", message), log.Any("before", message.BeforeDelete))
	reply, found := msgSet.Get(message.ID)
	if !found {
		log.S(log.Debug, "message not handled before", log.Any("id", message.ID))
		return
	}
	log.S(log.Info, "message delete detected",
		log.Any("id", message.ID),
		log.Any("reply", reply),
		log.Any("before", message.BeforeDelete))
	handleMessage(session, &discordgo.MessageCreate{Message: message.Message}, reply)
}

func registerCommands(session *discordgo.Session) {
	command := &discordgo.ApplicationCommand{
		Name:        "grol",
		Description: "Information about GROL",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command",
				Description: "Get information about GROL",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "help",
						Value: "help",
					},
					{
						Name:  "version",
						Value: "version",
					},
					{
						Name:  "source",
						Value: "source",
					},
					{
						Name:  "bug",
						Value: "bug",
					},
				},
			},
		},
	}

	_, err := session.ApplicationCommandCreate(session.State.User.ID, "", command)
	if err != nil {
		log.Fatalf("Cannot create command: %v", err)
	}
}

func interactionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		log.LogVf("Ignoring non command interaction type: %v", interaction.Type)
		return
	}
	for _, option := range interaction.ApplicationCommandData().Options {
		serverName := "DM"
		channelName := "DM"
		userName := ""
		server := interaction.GuildID
		if server != "" { //nolint:nestif // TODO share with in handleMessage
			channel, err := session.State.Channel(interaction.ChannelID)
			if err != nil {
				log.S(log.Error, "unable to get channel info", log.Any("err", err))
				channelName = Unknown
			} else {
				channelName = channel.Name
			}
			svr, err := session.State.Guild(interaction.GuildID)
			if err != nil {
				log.S(log.Error, "unable to get server info", log.Any("err", err))
				serverName = Unknown
			} else {
				serverName = svr.Name
			}
			userName = interaction.Member.User.Username
		} else {
			userName = interaction.User.Username
		}
		log.S(log.Info, "interaction",
			log.Any("from", userName),
			log.Any("server", serverName),
			log.Any("channel", channelName),
			log.Any("content", option))
		option := option.StringValue()
		responseMessage := eval(option, false, false)
		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: responseMessage,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}
		err := session.InteractionRespond(interaction.Interaction, response)
		if err != nil {
			log.Errf("Error responding to interaction: %v", err)
		}
	}
}
