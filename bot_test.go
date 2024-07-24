package main_test

import (
	"testing"
	"time"

	bot "grol.io/grol-discord-bot"
)

func TestUptime(t *testing.T) {
	delta := 100*time.Millisecond + 26*time.Hour + 42*time.Minute
	startTime := time.Now().Add(-delta)
	actual := bot.UptimeString(startTime)
	expected := "1d2h42m0.1s"
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
	delta = 23*time.Hour + 5*time.Minute
	startTime = time.Now().Add(-delta)
	actual = bot.UptimeString(startTime)
	expected = "23h5m0s"
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
	delta = 96*time.Hour - 100*time.Millisecond
	startTime = time.Now().Add(-delta)
	actual = bot.UptimeString(startTime)
	expected = "3d23h59m59.9s"
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
}
