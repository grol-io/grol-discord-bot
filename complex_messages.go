package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"fortio.org/log"
	"github.com/bwmarrin/discordgo"
	"grol.io/grol/object"
)

/*
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
*/

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionMessageComponent {
		customID := i.MessageComponentData().CustomID
		// Parse the customID to figure out which cell was clicked, e.g., "cell_0_1"
		// Update the board state based on the player and re-render the board
		log.Infof("Button clicked: %s", customID)
		// Example response:
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage, // Updates the message with new board state
			Data: &discordgo.InteractionResponseData{
				Content:    "Clicked " + customID,
				Components: []discordgo.MessageComponent{
					// Recreate the updated board here
				},
			},
		})
		if err != nil {
			log.Errf("Error responding to interaction: %v", err)
		}
	}
}

type MessageState struct {
	Session   *discordgo.Session
	ChannelID string
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
			msg := args[0].(object.Map).Unwrap(true)
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
			return object.String{Value: m.ID}
		},
	}
	return cmd.Name, cmd
}

/*
Basic working JSON example:

{"content":"A test...","components":[{"type":1,"components":[{"label":"Option 1","type":2},{"label":"Option 2","type":2}]}]}

Now handled functionally in discord_message.gr
*/
