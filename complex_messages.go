package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"fortio.org/log"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/eval"
	"grol.io/grol/object"
	"grol.io/grol/repl"
)

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		log.Infof("Ignoring interaction type: %v", i.Type)
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
	code := fmt.Sprintf("state[%q] = discordInteraction(state[%q],%q,%s)", i.Message.ID, i.Message.ID, userID, json)
	log.Infof("Running code: %s", code)
	cfg := replConfig()
	cfg.PreInput = func(state *eval.State) {
		st := MessageState{
			Session:     s,
			Interaction: i.Interaction,
		}
		name, fn := InteractionRespondFunction(&st)
		state.Extensions[name] = fn
	}
	res, errs, _ := repl.EvalStringWithOption(cfg, code)
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
}

func InteractionRespondFunction(st *MessageState) (string, object.Extension) {
	cmd := object.Extension{
		Name:       "InteractionRespond",
		MinArgs:    1,
		MaxArgs:    1,
		ArgTypes:   []object.Type{object.MAP},
		ClientData: st, // Unique to the current interpreter
		Callback: func(cdata any, _ string, args []object.Object) object.Object {
			msgContext, ok := cdata.(*MessageState)
			if !ok {
				log.Fatalf("Invalid client data type: %T", cdata)
			}
			log.Debugf("InteractionRespond Message state %+v", msgContext)
			msg := args[0].(object.Map).Unwrap(true).(map[string]any)
			endpoint := discordgo.EndpointInteractionResponse(msgContext.Interaction.ID, msgContext.Interaction.Token)
			_, err := msgContext.Session.RequestWithBucketID(http.MethodPost, endpoint, msg, endpoint)
			if err != nil {
				log.Errf("Error sending interaction response: %v", err)
				return object.Error{Value: fmt.Sprintf("Error sending interaction response: %v", err)}
			}
			return object.NULL
		},
	}
	return cmd.Name, cmd
}

func ChannelMessageSendComplexFunction(st *MessageState) (string, object.Extension) {
	cmd := object.Extension{
		Name:       "ChannelMessageSendComplex",
		MinArgs:    1,
		MaxArgs:    1,
		ArgTypes:   []object.Type{object.MAP},
		ClientData: st, // Unique to the current interpreter
		Callback: func(cdata any, _ string, args []object.Object) object.Object {
			msgContext, ok := cdata.(*MessageState)
			if !ok {
				log.Fatalf("Invalid client data type: %T", cdata)
			}
			log.Debugf("ChannelMessageSendComplex Message state %+v", msgContext)
			chID := msgContext.ChannelID
			msg := args[0].(object.Map).Unwrap(true).(map[string]any)
			// Make this a reply to identify the origin source (person) of the message.
			ref := make(map[string]string, 1)
			ref["message_id"] = msgContext.TriggerMessageID
			msg["message_reference"] = ref
			log.Debugf("Sending message to channel %s: %v", chID, msg)
			endpoint := discordgo.EndpointChannelMessages(chID)
			response, err := msgContext.Session.RequestWithBucketID(http.MethodPost, endpoint, msg, endpoint)
			if err != nil {
				log.Errf("Error sending message: %v", err)
				return object.Error{Value: fmt.Sprintf("Error sending message: %v", err)}
			}
			var m discordgo.Message
			err = json.Unmarshal(response, &m)
			if err != nil {
				log.Errf("Error unmarshalling message: %v", err)
				return object.Error{Value: fmt.Sprintf("Error unmarshalling message: %v", err)}
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
