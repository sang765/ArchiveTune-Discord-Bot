package discord

import "testing"

func TestParsePrefixCommandSolvedOnly(t *testing.T) {
	action, ok := ParsePrefixCommand(".solved")
	if !ok {
		t.Fatal("expected .solved to be recognized")
	}
	if action.TagName != "Solved" || action.TitlePrefix != "[SOLVED]" {
		t.Fatalf("unexpected action: %#v", action)
	}
	if _, ok := ParsePrefixCommand(".sloved"); ok {
		t.Fatal("expected .sloved to be removed as a valid command")
	}
}

func TestParsePrefixCommandFalseReport(t *testing.T) {
	action, ok := ParsePrefixCommand(".false")
	if !ok {
		t.Fatal("expected .false to be recognized")
	}
	if action.TagName != "False report" || action.TitlePrefix != "[FALSE REPORT]" {
		t.Fatalf("unexpected action: %#v", action)
	}
}

func TestGuessPrefixCommandCorrectsSloved(t *testing.T) {
	action, command, ok := GuessPrefixCommand(".sloved", 2)
	if !ok {
		t.Fatal("expected .sloved to be recognized as an unambiguous typo")
	}
	if command != ".solved" || action.TagName != "Solved" {
		t.Fatalf("unexpected correction: command=%q action=%#v", command, action)
	}
}

func TestGuessPrefixCommandRejectsArguments(t *testing.T) {
	if _, _, ok := GuessPrefixCommand(".solved extra", 2); ok {
		t.Fatal("expected command with arguments to be rejected")
	}
}
