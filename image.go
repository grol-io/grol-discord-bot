package main

import (
	"bytes"
	"image/png"

	"fortio.org/log"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/object"
)

func SendImageFunction(st *MessageState) (string, object.Extension) {
	cmd := object.Extension{
		Name:       "SendImage",
		MinArgs:    1,
		MaxArgs:    1,
		ArgTypes:   []object.Type{object.STRING},
		ClientData: st, // Unique to the current interpreter
		Callback: func(cdata any, _ string, args []object.Object) object.Object {
			msgContext, ok := cdata.(*MessageState)
			if !ok {
				log.Fatalf("Invalid client data type: %T", cdata)
			}
			log.Debugf("SendImage Message state %+v", msgContext)
			chID := msgContext.ChannelID
			imgName := args[0].(object.String).Value
			img, ok := msgContext.ImageMap[args[0]]
			if !ok {
				return object.Errorf("image not found: %q", imgName)
			}
			buf := bytes.Buffer{}
			_ = png.Encode(&buf, img.Image)
			msg := discordgo.MessageSend{File: &discordgo.File{Name: args[0].(object.String).Value + ".png", Reader: &buf}}
			msg.AllowedMentions = &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			}
			msg.Reference = &discordgo.MessageReference{MessageID: msgContext.TriggerMessageID}
			response, err := msgContext.Session.ChannelMessageSendComplex(chID, &msg)
			log.Debugf("Sending image to channel %s: %v", chID, msg)
			if err != nil {
				log.Errf("Error sending message: %v", err)
				return object.Errorf("Error sending message: %v", err)
			}
			updateMap(msgContext.TriggerMessageID, response.ID)
			return object.String{Value: response.ID}
		},
	}
	return cmd.Name, cmd
}
