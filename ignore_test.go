package main

import (
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
)

var ignoreTestMu sync.Mutex

func withIgnoredUsers(t *testing.T, list string) {
	t.Helper()
	ignoreTestMu.Lock()
	ignoreMu.RLock()
	prev := make([]string, 0, len(ignored))
	for userID := range ignored {
		prev = append(prev, userID)
	}
	ignoreMu.RUnlock()
	SetIgnoredUsers(list)
	t.Cleanup(func() {
		SetIgnoredUsers("")
		for _, userID := range prev {
			AddIgnoredUser(userID)
		}
		ignoreTestMu.Unlock()
	})
}

func TestSetIgnoredUsers(t *testing.T) {
	withIgnoredUsers(t, " 123 : :<@!456> ")
	if !IsIgnored("123") {
		t.Fatal("expected 123 to be ignored")
	}
	if !IsIgnored("456") {
		t.Fatal("expected 456 to be ignored")
	}
	if IsIgnored("789") {
		t.Fatal("did not expect 789 to be ignored")
	}
	if got := IgnoredUsersString(); got != "123:456" {
		t.Fatalf("expected ignored users list to be sorted, got %q", got)
	}
}

func TestAddIgnoredUser(t *testing.T) {
	withIgnoredUsers(t, "")
	if !AddIgnoredUser("<@!789>") {
		t.Fatal("expected first add to report new user")
	}
	if !IsIgnored("789") {
		t.Fatal("expected normalized user id to be ignored")
	}
	if AddIgnoredUser("789") {
		t.Fatal("expected duplicate add to report existing user")
	}
}

func TestRemoveIgnoredUser(t *testing.T) {
	withIgnoredUsers(t, "123:<@!789>")
	if !RemoveIgnoredUser("<@789>") {
		t.Fatal("expected existing user to be removed")
	}
	if IsIgnored("789") {
		t.Fatal("expected normalized user id to be removed from ignore list")
	}
	if RemoveIgnoredUser("789") {
		t.Fatal("expected removing an already-removed user to report missing user")
	}
	if !IsIgnored("123") {
		t.Fatal("expected other ignored users to remain ignored")
	}
}

func TestEvalInputIgnoreCommandRequiresAdmin(t *testing.T) {
	withIgnoredUsers(t, "")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "7"},
		},
	}
	got := evalInput("ignore 99", p)
	want := "⛔️ Only the bot admin can ignore users, please ask <@42>"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if IsIgnored("99") {
		t.Fatal("non-admin should not be able to ignore users")
	}
}

func TestEvalInputIgnoreCommandAddsUser(t *testing.T) {
	withIgnoredUsers(t, "")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "42"},
		},
	}
	got := evalInput("ignore <@!99>", p)
	want := "🙈 Ignoring messages from <@99>."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if !IsIgnored("99") {
		t.Fatal("admin ignore command should add the user to the ignore list")
	}
}

func TestEvalInputUnignoreCommandRequiresAdmin(t *testing.T) {
	withIgnoredUsers(t, "99")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "7"},
		},
	}
	got := evalInput("unignore 99", p)
	want := "⛔️ Only the bot admin can unignore users, please ask <@42>"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if !IsIgnored("99") {
		t.Fatal("non-admin should not be able to unignore users")
	}
}

func TestEvalInputUnignoreCommandShowsUsage(t *testing.T) {
	withIgnoredUsers(t, "99")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "42"},
		},
	}
	got := evalInput("unignore", p)
	want := "💡 Usage: `!grol unignore <userid>`"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEvalInputUnignoreCommandRemovesUser(t *testing.T) {
	withIgnoredUsers(t, "99")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "42"},
		},
	}
	got := evalInput("unignore <@!99>", p)
	want := "🙉 No longer ignoring messages from <@99>."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if IsIgnored("99") {
		t.Fatal("admin unignore command should remove the user from the ignore list")
	}
}

func TestEvalInputUnignoreCommandRejectsInvalidUserID(t *testing.T) {
	withIgnoredUsers(t, "")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "42"},
		},
	}
	got := evalInput("unignore not-a-user", p)
	want := "💡 Usage: `!grol unignore <userid>`"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEvalInputIgnoreCommandRejectsInvalidUserID(t *testing.T) {
	withIgnoredUsers(t, "")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	p := &CommandParams{
		message: &discordgo.Message{
			Author: &discordgo.User{ID: "42"},
		},
	}
	got := evalInput("ignore not-a-user", p)
	want := "💡 Usage: `!grol ignore <userid>`"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if IgnoredUsersCount() != 0 {
		t.Fatal("invalid user ids should not be added to the ignore list")
	}
}

func TestEvalInputIgnoreCommandWithoutMessageDoesNotPanic(t *testing.T) {
	withIgnoredUsers(t, "")
	prevAdmin := BotAdmin
	BotAdmin = "42"
	t.Cleanup(func() {
		BotAdmin = prevAdmin
	})
	got := evalInput("ignore 99", &CommandParams{})
	want := "⛔️ Only the bot admin can ignore users, please ask <@42>"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInteractionUserID(t *testing.T) {
	t.Run("user", func(t *testing.T) {
		got, ok := interactionUserID(&discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				User: &discordgo.User{ID: "123"},
			},
		})
		if !ok || got != "123" {
			t.Fatalf("expected direct user id, got %q, %t", got, ok)
		}
	})
	t.Run("member user", func(t *testing.T) {
		got, ok := interactionUserID(&discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "456"},
				},
			},
		})
		if !ok || got != "456" {
			t.Fatalf("expected member user id, got %q, %t", got, ok)
		}
	})
	t.Run("missing user", func(t *testing.T) {
		got, ok := interactionUserID(&discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{},
		})
		if ok || got != "" {
			t.Fatalf("expected missing user result, got %q, %t", got, ok)
		}
	})
}
