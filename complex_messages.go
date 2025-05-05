package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"

	"fortio.org/log"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/eval"
	"grol.io/grol/extensions"
	"grol.io/grol/object"
	"grol.io/grol/repl"
)

func errorReply(s *discordgo.Session, i *discordgo.InteractionCreate, userID, msg string) {
	log.S(log.Warning, msg, log.Any("author", userID))
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🔴 I'm sorry Dave. I'm afraid I can't do that.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
	err := s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		log.Errf("Error responding to interaction: %v", err)
	}
}

func processApplicationCommandInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.S(log.Info, "Processing application command interaction", log.Any("interaction", i))
	if i.ApplicationCommandData().Options != nil {
		slashCmdInteraction(s, i)
		return
	}
	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}
	resolved := i.ApplicationCommandData().Resolved
	if resolved == nil {
		errorReply(s, i, userID, "Resolved data is nil")
		return
	}
	msgs := resolved.Messages
	if len(msgs) != 1 {
		errorReply(s, i, userID, "Expected exactly one message")
		return
	}
	var msg *discordgo.Message
	for _, v := range msgs {
		msg = v
	}
	if !IsThisBot(msg.Author.ID) {
		errorReply(s, i, userID, "Expected message to be from the bot")
		return
	}
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✨ Poof! The message has vanished into the void.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
	log.S(log.Warning, "command-delete", log.Any("user", userID), log.Any("message", msg))
	err := s.ChannelMessageDelete(i.ChannelID, msg.ID)
	if err != nil {
		log.Errf("Error deleting message: %v", err)
	}
	err = s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		log.Errf("Error responding to interaction: %v", err)
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		processApplicationCommandInteraction(s, i)
		return
	}
	if i.Type != discordgo.InteractionMessageComponent {
		log.S(log.Info, "Ignoring interaction", log.Any("type", i.Type))
		return
	}
	log.S(log.Info, "interaction", log.Any("interaction", i))
	// Call into grol interpreter.
	data := i.MessageComponentData()
	json, err := json.Marshal(data)
	if err != nil {
		log.Critf("Error marshaling interaction data: %v", err)
		return
	}
	// Key state of the message ID.
	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}
	code := fmt.Sprintf("discord.doInteraction(%q,%q,%s)", i.Message.ID, userID, json)
	log.Infof("Running code: %s", code)
	cfg := replConfig()
	cfg.PreInput = func(state *eval.State) {
		st := MessageState{
			Session:     s,
			Interaction: i.Interaction,
			ImageMap:    state.Extensions["image.new"].ClientData.(extensions.ImageMap),
		}
		name, fn := InteractionRespondFunction(&st)
		state.Extensions[name] = fn
	}
	res, errs, _ := repl.EvalStringWithOption(context.Background(), cfg, code)
	log.Infof("Interaction (ignored) result: %q errs %v", res, errs)
	if len(errs) > 0 {
		p := &CommandParams{
			session:   s,
			message:   i.Message,
			channelID: i.ChannelID,
		}
		res += "<@" + userID + ">:" + errorsBlock(errs)
		reply(s, res, p)
	}
}

type MessageState struct {
	Session          *discordgo.Session
	ChannelID        string
	TriggerMessageID string
	// for interaction responses
	Interaction *discordgo.Interaction
	ImageMap    extensions.ImageMap
}

func AddImage(st *MessageState, msg *discordgo.MessageSend, args []object.Object) object.Object {
	if len(args) <= 1 {
		return object.NULL
	}
	imgName := args[1].(object.String).Value
	img, ok := st.ImageMap[args[1]]
	if !ok {
		return object.Errorf("image not found: %q", imgName)
	}
	buf := bytes.Buffer{}
	_ = png.Encode(&buf, img.Image)
	msg.File = &discordgo.File{Name: imgName + ".png", Reader: &buf}
	return object.NULL
}

func InteractionRespondFunction(st *MessageState) (string, object.Extension) {
	cmd := object.Extension{
		Name:       "InteractionRespond",
		MinArgs:    1,
		MaxArgs:    2,
		Help:       "Respond and optionally attach an image",
		ArgTypes:   []object.Type{object.MAP, object.STRING},
		ClientData: st, // Unique to the current interpreter
		Callback: func(cdata any, _ string, args []object.Object) object.Object {
			msgContext, ok := cdata.(*MessageState)
			if !ok {
				log.Fatalf("Invalid client data type: %T", cdata)
			}
			log.Debugf("InteractionRespond Message state %+v", msgContext)
			ir := MsgMapToInteractionResponse(args[0].(object.Map))
			if ir == nil {
				// already logged
				return object.Errorf("Error converting map to struct")
			}
			dms := discordgo.MessageSend{}
			oerr := AddImage(msgContext, &dms, args)
			if oerr != object.NULL {
				log.Errf("Error adding image: %v", oerr)
				return oerr
			}
			if dms.File != nil {
				ir.Data.Files = []*discordgo.File{dms.File}
				// Explicitly set empty attachments to replace existing ones
				ir.Data.Attachments = &[]*discordgo.MessageAttachment{}
				log.LogVf("Added file to interaction response: %s", dms.File.Name)
			}
			err := msgContext.Session.InteractionRespond(msgContext.Interaction, ir)
			if err != nil {
				log.Errf("Error sending interaction response: %v", err)
				return object.Errorf("Error sending interaction response: %v", err)
			}
			return object.NULL
		},
	}
	return cmd.Name, cmd
}

func MsgMapToInteractionResponse(msg object.Map) *discordgo.InteractionResponse {
	dm := discordgo.Message{}
	ir := discordgo.InteractionResponseData{}
	dataPart, found := msg.Get(object.String{Value: "data"})
	if !found {
		log.Errf("No \"data\" part found in map")
		return nil
	}
	err := extensions.MapToStruct(dataPart.(object.Map), &dm) // MessageSend is lacking the UnmarshalJSON
	if err != nil {
		log.Errf("Error converting map to struct: %v", err)
		return nil
	}
	ir.Content = dm.Content
	ir.Components = dm.Components
	ir.AllowedMentions = &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{},
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &ir,
	}
}

func MsgMapToMessageSend(msg object.Map) *discordgo.MessageSend {
	dm := discordgo.Message{}
	dms := discordgo.MessageSend{}
	err := extensions.MapToStruct(msg, &dm) // MessageSend is lacking the UnmarshalJSON
	if err != nil {
		log.Errf("Error converting map to struct: %v", err)
		return nil
	}
	dms.Content = dm.Content
	dms.Components = dm.Components
	return &dms
}

func ChannelMessageSendComplexFunction(st *MessageState) (string, object.Extension) {
	cmd := object.Extension{
		Name:       "ChannelMessageSendComplex",
		MinArgs:    1,
		MaxArgs:    2,
		ArgTypes:   []object.Type{object.MAP, object.STRING},
		ClientData: st, // Unique to the current interpreter
		Callback: func(cdata any, _ string, args []object.Object) object.Object {
			msgContext, ok := cdata.(*MessageState)
			if !ok {
				log.Fatalf("Invalid client data type: %T", cdata)
			}
			log.LogVf("ChannelMessageSendComplex Message state %+v", msgContext)
			chID := msgContext.ChannelID
			dms := MsgMapToMessageSend(args[0].(object.Map))
			if dms == nil {
				// already logged
				return object.Errorf("Error converting map to struct")
			}
			// Make this a reply to identify the origin source (person) of the message.
			dms.Reference = &discordgo.MessageReference{MessageID: msgContext.TriggerMessageID}
			dms.AllowedMentions = &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			}
			oerr := AddImage(msgContext, dms, args)
			if oerr != object.NULL {
				log.Errf("Error adding image: %v", oerr)
				return oerr
			}
			log.LogVf("Sending message to channel %s: %v", chID, dms)
			// putting it back in discordgo.MessageSend{}
			m, err := msgContext.Session.ChannelMessageSendComplex(chID, dms)
			if err != nil {
				log.Errf("Error sending message: %v", err)
				return object.Errorf("Error sending message: %v", err)
			}
			updateMap(msgContext.TriggerMessageID, m.ID)
			return object.String{Value: m.ID}
		},
	}
	return cmd.Name, cmd
}

/*
Basic working JSON example:

{"content":"A test...","components":[{"type":1,"components":[{"label":"Option 1","type":2},{"label":"Option 2","type":2}]}]}

Now handled in discord.gr
*/
