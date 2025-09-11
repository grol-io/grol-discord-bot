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
	actual = bot.DurationString(delta)
	expected = "23h5m"
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
	delta = 96*time.Hour - 100*time.Millisecond
	actual = bot.DurationString(delta)
	expected = "3d23h59m59.9s"
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
	delta = 8*24*time.Hour + 3*time.Minute
	actual = bot.DurationString(delta)
	expected = "1w1d3m"
	if actual != expected {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
}

func TestRemoveBackticks(t *testing.T) {
	// table driven inpute,expected
	tests := []struct {
		input, expected string
	}{
		{"   foo   \n   bar   ", "foo   \n   bar"},
		{
			`
this is before code
` + "```go" + `
a=1
b=2
` + "```" + `
some stuff after code`,
			"a=1\nb=2",
		},
		{
			`
this is before code
` + "```go" + `
a=1
b=2
` + "```" + `
some stuff after code
` + "```c=3``` and ```d=4```",
			"a=1\nb=2\nc=3\nd=4",
		},
	}
	for _, test := range tests {
		actual := bot.RemoveTripleBackticks(test.input)
		if actual != test.expected {
			t.Errorf("---For---\n%s\n---Expected %q, but got %q", test.input, test.expected, actual)
		}
	}
}

func TestSmartQuotesToRegular(t *testing.T) {
	// table driven inpute,expected
	tests := []struct {
		input, expected string
	}{
		{`abc“def`, `abc"def`},
		{"   no quotes  ", "   no quotes  "},
		{
			"“this is a quote”",
			`"this is a quote"`,
		},
		{
			`\“”`,
			`\“”`,
		},
		{
			`len("“")`,
			`len("“")`,
		},
		{
			`load(“”)`,
			`load("")`,
		},
		{
			"println(“this is a quote”); println(“this is another quote”)",
			`println("this is a quote"); println("this is another quote")`,
		},
	}
	for _, test := range tests {
		actual := bot.SmartQuotesToRegular(test.input)
		if actual != test.expected {
			t.Errorf("---For---\n%s\n---Expected %q, but got %q", test.input, test.expected, actual)
		}
	}
}
