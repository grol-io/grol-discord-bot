package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"fortio.org/log"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/object"
)

func sendTicTacToeBoard(s *discordgo.Session, channelID string, board [3][3]string) {
	components := []discordgo.MessageComponent{}
	for i := range 3 {
		row := discordgo.ActionsRow{}
		for j := range 3 {
			label := board[i][j]
			if label == "" {
				label = "\u200B" // zero width space
			}
			row.Components = append(row.Components, discordgo.Button{
				Label:    label,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("cell_%d_%d", i, j),
			})
		}
		components = append(components, row)
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:    "Tic-Tac-Toe",
		Components: components,
	})
	if err != nil {
		log.Errf("Error sending board message: %v", err)
	}
}

func sendButtonMessage(s *discordgo.Session, channelID string) {
	message := &discordgo.MessageSend{
		Content: "Choose an option:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Option 1",
						Style:    discordgo.PrimaryButton,
						CustomID: "button_1",
					},
					discordgo.Button{
						Label:    "Option 2",
						Style:    discordgo.SecondaryButton,
						CustomID: "button_2",
					},
				},
			},
		},
	}
	log.S(log.Info, "Sending button message", log.Any("message", message))
	_, err := s.ChannelMessageSendComplex(channelID, message)
	if err != nil {
		log.Errf("Error sending button message: %v", err)
	}
}

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionMessageComponent {
		customID := i.MessageComponentData().CustomID
		// Parse the customID to figure out which cell was clicked, e.g., "cell_0_1"
		// Update the board state based on the player and re-render the board
		log.Infof("Button clicked: %s", customID)
		// Example response:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage, // Updates the message with new board state
			Data: &discordgo.InteractionResponseData{
				Content:    "Updated Tic-Tac-Toe",
				Components: []discordgo.MessageComponent{
					// Recreate the updated board here
				},
			},
		})
	}
}

func OnInteractionCreate2(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionMessageComponent {
		switch i.MessageComponentData().CustomID {
		case "button_1":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You clicked Option 1!",
				},
			})
		case "button_2":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You clicked Option 2!",
				},
			})
		}
	}
}

/*
"Components":[{"Components":{"Label":"Foo","type":2} , "type":1}]

	func discordMessage(chanID){
		msg = {"Content":"A test...",
			"Components": [{"ActionsRow": [{"Components": [{"Label": "Option 1"}, {"Label": "Option 2"}]}]}]
		}
		ChannelMessageSendComplex(chanID,msg)
	}
*/
func AddGrolCommands(session *discordgo.Session) {
	cmd := object.Extension{
		Name:     "ChannelMessageSendComplex",
		MinArgs:  2,
		MaxArgs:  2,
		ArgTypes: []object.Type{object.STRING, object.MAP},
		Callback: func(_ any, _ string, args []object.Object) object.Object {
			chID := args[0].(object.String).Value
			msg := args[1].(object.Map)
			var buf bytes.Buffer
			err := msg.JSON(&buf)
			if err != nil {
				log.Errf("Error marshalling JSON: %v", err)
				return object.Error{Value: fmt.Sprintf("Error marshalling JSON: %v", err)}
			}
			msgJSON := buf.Bytes()
			var data *discordgo.MessageSend
			err = json.Unmarshal(msgJSON, &data)
			if err != nil {
				log.Errf("Error unmarshalling JSON %q: %v", msgJSON, err)
				return object.Error{Value: fmt.Sprintf("Error unmarshalling JSON %q: %v", msgJSON, err)}
			}
			_, err = session.ChannelMessageSendComplex(chID, data)
			if err != nil {
				log.Errf("Error sending message: %v", err)
				return object.Error{Value: fmt.Sprintf("Error sending message: %v", err)}
			}
			return object.NULL
		},
	}
	_ = object.CreateFunction(cmd)
}
