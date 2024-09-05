package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/version"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol-discord-bot/fixedmap"
	"grol.io/grol/eval"
	"grol.io/grol/extensions"
	"grol.io/grol/repl"
)

var (
	BotToken string
	AutoSave bool
	BotAdmin string
	// State for edit to replies.
	msgSet       *fixedmap.FixedMap[string, string]
	botStartTime time.Time
	selfID       string // This bot's user ID.
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
	session.AddHandler(deleteMessage)
	session.AddHandler(interactionCreate)
	session.AddHandler(messageReactionAdd)

	// open session
	err = session.Open()
	if err != nil {
		log.Fatalf("Init discordgo.Open error: %v", err)
	}
	defer session.Close() // close session, after function termination

	selfID = session.State.User.ID

	registerCommands(session)

	// Eval the library and save it.
	opts := repl.EvalStringOptions()
	opts.AutoSave = true // force saving the library to compact form even if autosave is off for user messages.
	res, errs, _ := repl.EvalStringWithOption(context.Background(), opts, libraryCode)
	if len(errs) > 0 {
		log.S(log.Critical, "Errors in library eval", log.Any("errors", errs))
	}
	log.S(log.Info, "Library eval result", log.String("result", res))

	log.Infof("Bot is now running with AutoSave=%t, BotAdmin=%s - Press CTRL-C or SIGTERM to exit.", AutoSave, BotAdmin)
	// keep bot running until there is NO os interruption (ctrl + C)
	cli.UntilInterrupted()
	err = session.Close()
	if err != nil {
		log.Errf("Error closing session: %v", err)
	}
	log.Infof("Bot is now stopped and exiting.")
}

func IsThisBot(id string) bool {
	return id == selfID
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

func setFormatMode(message *discordgo.Message, p *CommandParams) {
	message.Content = tagToCmd(message.Content, selfID)
	p.formatMode = strings.HasPrefix(message.Content, formatModeStr)
	p.compactMode = strings.HasPrefix(message.Content, compactModeStr)
	p.verbatimMode = strings.HasPrefix(message.Content, verbatimModeStr)
	p.debugParenMode = strings.HasPrefix(message.Content, debugParenStr)
	switch {
	case p.formatMode:
		message.Content = message.Content[len(formatModeStr):]
	case p.compactMode:
		message.Content = message.Content[len(compactModeStr):]
	case p.verbatimMode:
		message.Content = message.Content[len(verbatimModeStr):]
	case p.debugParenMode:
		message.Content = message.Content[len(debugParenStr):]
	}
	message.Content = strings.TrimPrefix(message.Content, grolPrefix)
}

func handleDM(session *discordgo.Session, message *discordgo.Message, replyID string) {
	log.S(log.Info, "direct-message",
		log.Any("from", message.Author.Username),
		log.Any("content", message.Content))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	p := &CommandParams{
		session:   session,
		message:   message,
		channelID: message.ChannelID,
		replyID:   replyID,
		useReply:  false,
	}
	setFormatMode(message, p)
	replyID = evalAndReply(session, "dm-reply", message.Content, p)
	updateMap(message.ID, replyID)
}

var (
	growlVersion, _, _ = version.FromBuildInfoPath("grol.io/grol")
	grolPrefix         = "!grol"
	formatModeStr      = grolPrefix + " -f"
	compactModeStr     = grolPrefix + " -c"
	verbatimModeStr    = grolPrefix + " -v"
	debugParenStr      = grolPrefix + " -d"
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
		s = strings.TrimPrefix(s, "```js")
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

const (
	smartQuotes   = "“”"                 // 2 of the smart double quotes here
	smartQuoteLen = len(smartQuotes) / 2 // 3 bytes each
)

func SmartQuotesToRegular(s string) string {
	idx := strings.IndexAny(s, smartQuotes)
	// If not found or just after a regular double quote or escaped by \
	// then keep as is (short cut).
	if idx == -1 || (idx > 0 && (s[idx-1] == '"' || s[idx-1] == '\\')) {
		return s
	}
	buf := make([]byte, 0, len(s)-smartQuoteLen+1) // smart quotes are 3 bytes each
	buf = append(buf, s[:idx]...)
	replaceQuote := func(rel int) {
		buf = append(buf, s[idx:idx+rel]...)
		buf = append(buf, '"')
		idx += rel + smartQuoteLen
	}
	replaceQuote(0) // Replace the first smart quote
	for {
		rel := strings.IndexAny(s[idx:], smartQuotes)
		if rel == -1 {
			buf = append(buf, s[idx:]...)
			break
		}
		replaceQuote(rel)
	}
	return string(buf)
}

func replConfig() repl.Options {
	cfg := repl.EvalStringOptions()
	cfg.AutoLoad = true
	cfg.AutoSave = AutoSave
	cfg.MaxDepth = *depth
	cfg.MaxValueLen = *maxLen
	cfg.PanicOk = *panicF
	cfg.MaxDuration = *maxDur
	return cfg
}

// TODO: switch to an option/config object and maybe an enum as verbatim and compact and format are all exclusive.
func evalInput(input string, p *CommandParams) string {
	var res string
	input = strings.TrimSpace(input) // we do it again so "   !grol    help" works
	switch input {
	case "", "help", "-h", "--help", "-help":
		res = "💡 Grol bot help: grol bot evaluates [grol](<https://grol.io>) language fragments, as simple as expressions like `1+1`" +
			" and as complex as defining closures, using map, arrays, etc... the syntax is similar to go (without needing " +
			"`:=`, plain `=` is enough). Use `info` to see all functions, keywords, etc..." +
			" Try `TicTacToe()` or `Butterfly()` for more advanced examples that includes grol handling discord interactions and complex messages.\n\n" +
			"Either in DM or @grol or with `!grol` prefix (or `!grol -f` for also showing formatted code, `-c` in compact mode," +
			" `-d` debug expressions)" +
			" in a channel, you can type any grol code and the bot will evaluate it (only code blocks if there are any).\n" +
			"Tip: _You can **edit** your messages and I will re-run and edit mine (less floody that way). Thank you!_\n\n" +
			"Also supported `!grol version`, `!grol source`, `!grol buildinfo`, `!grol bug`.\n" +
			"You can also try the /grol command, answers will be visible only to you!"
	case "source":
		res = "📄 [github.com/grol-io/grol-discord-bot](<https://github.com/grol-io/grol-discord-bot>)" +
			" and [grol-io](<https://grol.io>)"
	case "uptime", "version", "--version", "-version":
		res = "📦 Grol bot version: " + cli.ShortVersion + ", `grol` language version " + growlVersion +
			" ⏰ Uptime: " + UptimeString(botStartTime)
	case "buildinfo":
		res = "📦ℹ️```" + cli.FullVersion + "```"
	case "bug":
		res = "🐞 Please report any issue or suggestion at " +
			"[github.com/grol-io/grol-discord-bot/issues](<https://github.com/grol-io/grol-discord-bot/issues>)"
	case "reset":
		if !IsAdmin(p.message.Author.ID) {
			return errorsBlock([]string{"Only the bot admin can reset the bot - please ask <@" + BotAdmin + ">"})
		}
		log.Critf("Admin %s requested reset", p.message.Author.ID)
		scheduleReset(p.session)
		return "🔄 Resetting bot per <@" + BotAdmin + ">, brb!."
	default:
		// TODO: stdout vs stderr vs result. https://github.com/grol-io/grol/issues/33
		//   !grol
		//   ```go
		//   1+1
		//   ```
		//   look at the result of 1+1
		// in a single message and not get errors on the extra text (meanwhile, add //).
		input = RemoveTripleBackticks(input)
		cfg := replConfig()
		cfg.Compact = p.compactMode
		cfg.AllParens = p.debugParenMode
		cfg.PreInput = func(state *eval.State) {
			st := MessageState{
				Session:          p.session,
				ChannelID:        p.channelID,
				TriggerMessageID: p.message.ID,
				ImageMap:         state.Extensions["image"].ClientData.(extensions.ImageMap),
			}
			name, fn := ChannelMessageSendComplexFunction(&st)
			state.Extensions[name] = fn
			name, fn = SendImageFunction(&st)
			state.Extensions[name] = fn
		}
		// Turn smart quotes back into regular quotes - https://github.com/grol-io/grol-discord-bot/issues/57
		input = SmartQuotesToRegular(input)
		evalres, errs, formatted := repl.EvalStringWithOption(context.Background(), cfg, input)
		if (p.formatMode || p.compactMode || p.debugParenMode) && formatted != "" {
			res = formatModeStr
			if p.compactMode {
				res = compactModeStr
			}
			if p.debugParenMode {
				res = debugParenStr
			}
			res += "\n```go\n" + formatted + "``` produces: "
		}
		evalres = strings.TrimSpace(evalres)
		p.hasErrors = len(errs) > 0
		if !p.hasErrors {
			if evalres == "" {
				evalres = "nil"
			}
			if p.verbatimMode {
				return evalres
			}
		}
		if evalres != "" {
			res += "```go\n" + evalres + "\n```\n"
		}
		if p.hasErrors {
			res += errorsBlock(errs)
		}
	}
	return res
}

func errorsBlock(errs []string) string {
	res := "```diff"
	for i, e := range errs {
		if i >= 2 {
			n := len(errs) - i
			res += fmt.Sprintf("\n...%d more %s...", n, cli.Plural(n, "error"))
			break
		}
		res += "\n-\t" + strings.Join(strings.Split(e, "\n"), "\n-\t")
	}
	res += "\n```Tip: _You can **edit** your message to correct instead of making a new one!_"
	return res
}

// Discord's limit - some margin for that adding we are truncating, in characters/runes.
const MaxMessageLengthInRunes = 2000 - 100

type CommandParams struct {
	session   *discordgo.Session
	message   *discordgo.Message // Message being replied to/processed.
	channelID string             // shortcut for message.ChannelID or id for a DM.
	// If we already replied and have an ID of that reply (to edit it).
	replyID string
	// Formatting options. useReply selects if we should use reply (in channel) or send (DMs).
	formatMode, compactMode, debugParenMode, verbatimMode, useReply, hasErrors bool
}

// returns the id of the reply.
func evalAndReply(session *discordgo.Session, info, input string, p *CommandParams) string {
	res := evalInput(input, p)
	level := log.Info
	msg := "response"
	runes := []rune(res)
	if len(runes) > MaxMessageLengthInRunes {
		res = string(runes[:MaxMessageLengthInRunes]) +
			fmt.Sprintf("```...truncated from %d characters (%d bytes)...", len(runes), len(res))
		level = log.Warning
		msg = "truncated response"
	}
	log.S(level, info, log.String(msg, res))
	return reply(session, res, p)
}

func reply(session *discordgo.Session, response string, p *CommandParams) string {
	var err error
	useEdit := p.replyID != ""
	if !useEdit && !p.hasErrors { // if there was an error despite the previous interaction, do a reply anyway.
		reply, found := msgSet.Get(p.message.ID)
		if found {
			log.S(log.Info, "Found previous reply (interaction) skipping reply", log.Any("reply", reply), log.Any("response", response))
			return reply
		}
	}
	if useEdit {
		// Edit of previous message case.
		_, err = session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:      p.replyID,
			Channel: p.channelID,
			Content: &response,
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			},
		})
		if err != nil {
			log.S(log.Error, "edit-error", log.Any("err", err))
		}
		return p.replyID
	}
	// New DM or new channel message cases.
	msg := &discordgo.MessageSend{
		Content: response,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	}
	if p.useReply {
		msg.Reference = &discordgo.MessageReference{
			MessageID: p.message.ID,
			ChannelID: p.message.ChannelID,
			GuildID:   p.message.GuildID,
		}
	}
	reply, err := session.ChannelMessageSendComplex(p.channelID, msg)
	if reply != nil {
		p.replyID = reply.ID
	}
	if err != nil {
		log.S(log.Error, "error", log.Any("err", err))
	}
	return p.replyID
}

func newMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	/* prevent bot responding to its own message
	this is achieved by looking into the message author id
	if message.author.id is same as bot.author.id then just return
	*/
	log.S(log.Debug, "message", log.Any("message", message))
	if IsThisBot(message.Author.ID) {
		return
	}
	handleMessage(session, message.Message, "")
}

func tagToCmd(msg, id string) string {
	return strings.ReplaceAll(msg, "<@"+id+">", "!grol")
}

func handleMessage(session *discordgo.Session, message *discordgo.Message, replyID string) {
	isDM := message.GuildID == ""
	message.Content = strings.TrimSpace(message.Content)
	if isDM {
		handleDM(session, message, replyID)
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
	info := "channel-message"
	mentioned := false
	for _, mention := range message.Mentions {
		if IsThisBot(mention.ID) {
			ref := message.ReferencedMessage
			if ref != nil {
				log.S(log.Info, "Ignoring bot mention in a reply", log.Any("ref", ref), log.Any("message", message))
				return
			}
			info = "channel-mention"
			mentioned = true
			message.Content = tagToCmd(message.Content, mention.ID)
			break
		}
	}
	if !mentioned && !strings.HasPrefix(message.Content, grolPrefix) {
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
	p := &CommandParams{
		session:   session,
		message:   message,
		channelID: message.ChannelID,
		replyID:   replyID,
		useReply:  true,
	}
	setFormatMode(message, p)
	log.S(log.Info, info,
		log.Any("from", message.Author.Username),
		log.Any("server", serverName),
		log.Any("channel", channelName),
		log.Any("content", message.Content),
		log.Bool("format", p.formatMode))
	if message.Author.Bot {
		log.S(log.Warning, "ignoring bot message", log.Any("message", message))
		return
	}
	replyID = evalAndReply(session, "channel-response", message.Content, p)
	updateMap(message.ID, replyID)
}

func updateMessage(session *discordgo.Session, message *discordgo.MessageUpdate) {
	log.S(log.Debug, "message update", log.Any("message", message))
	if IsThisBot(message.Author.ID) { // don't loop handling our own messages.
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
	handleMessage(session, message.Message, reply)
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
	handleMessage(session, message.Message, reply)
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
	_, err := session.ApplicationCommandCreate(selfID, "", command)
	if err != nil {
		log.Fatalf("Cannot create slash command: %v", err)
	}
	command = &discordgo.ApplicationCommand{
		Name: "grol: delete this",
		Type: discordgo.MessageApplicationCommand,
	}
	_, err = session.ApplicationCommandCreate(selfID, "", command)
	if err != nil {
		log.Fatalf("Cannot create chat command: %v", err)
	}
}

func slashCmdInteraction(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
		p := &CommandParams{
			session:      session,
			channelID:    interaction.ChannelID,
			replyID:      "",
			formatMode:   false,
			compactMode:  false,
			verbatimMode: false,
		}
		responseMessage := evalInput(option, p)
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

// This function will be called (due to AddHandler above) when a reaction is added to a message.
func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Fetch the message to check the author ID
	message, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		log.Errf("Error fetching message: %v", err)
		return
	}
	// Check if the message was ours
	if !IsThisBot(message.Author.ID) { // Directly use the session's bot ID
		log.Debugf("Reaction not on a message from the bot")
		return
	}
	user, err := s.User(r.UserID)
	if err != nil {
		log.Errf("Error fetching user: %v", err)
		return
	}
	log.S(log.Info, "reaction", log.Any("reaction", r), log.Any("user", user))
	if r.Emoji.Name == "👎" {
		log.S(log.Warning, "downvote-delete", log.Any("user", user), log.Any("message", message))
		err := s.ChannelMessageDelete(r.ChannelID, r.MessageID)
		if err != nil {
			log.Errf("Error deleting message: %v", err)
		}
	}
}

func IsAdmin(userID string) bool {
	if BotAdmin == "" {
		return false
	}
	return BotAdmin == userID
}

func scheduleReset(s *discordgo.Session) {
	go func() {
		registeredCommands, err := s.ApplicationCommands(selfID, "")
		if err != nil {
			log.Critf("Could not fetch registered commands: %v", err)
		}
		for _, v := range registeredCommands {
			err = s.ApplicationCommandDelete(selfID, "", v.ID)
			if err != nil {
				log.Critf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
		delay := 3 * time.Second
		log.Infof("All %d commands deleted, waiting %s before resetting", len(registeredCommands), delay)
		time.Sleep(delay)
		log.Critf("Resetting bot now")
		// unlink the .gr file to cleanup state.
		err = os.Rename(".gr", ".gr.bak")
		if err != nil {
			log.Critf("Error removing .gr file: %v", err)
		}
		// exit and get restarted by systemd.
		os.Exit(1)
	}()
}
