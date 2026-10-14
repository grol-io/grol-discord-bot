package main

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestTagToCmd(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"<@42> help", "!grol help"},
		{"<@!42> version", "!grol version"},
		{"<@42>", "!grol "},
		{"<@42> -f 1+1", "!grol -f 1+1"},
		{"<@42> !grol -c 1+1", "!grol -c 1+1"},
		{"please run this <@42>\n```go\n1+1\n```", "!grol please run this \n```go\n1+1\n```"},
		{"```1+1``` <@42>", "!grol ```1+1```"},
		{"no mention 1+1", "no mention 1+1"},
		{"<@43> other user", "<@43> other user"},
	}
	for _, test := range tests {
		actual := tagToCmd(test.input, "42")
		if actual != test.expected {
			t.Errorf("For %q expected %q, got %q", test.input, test.expected, actual)
		}
	}
}

func TestMentionCommands(t *testing.T) {
	old := selfID
	selfID = "42"
	defer func() { selfID = old }()
	for _, input := range []string{"<@42> help", "<@!42>   help", "  <@42>"} {
		msg := &discordgo.Message{Content: input}
		p := &CommandParams{message: msg}
		setFormatMode(msg, p)
		res := evalInput(msg.Content, p)
		if !strings.HasPrefix(res, "💡 Grol bot help") {
			t.Errorf("For %q expected help, got %q", input, res)
		}
	}
	msg := &discordgo.Message{Content: "<@42> -f 1+1"}
	p := &CommandParams{message: msg}
	setFormatMode(msg, p)
	if !p.formatMode || msg.Content != " 1+1" {
		t.Errorf("Expected format mode and \" 1+1\", got %v %q", p.formatMode, msg.Content)
	}
}
