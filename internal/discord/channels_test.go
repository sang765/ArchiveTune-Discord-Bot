package discord

import "testing"

func TestPrefixActionAllowedForChannel(t *testing.T) {
	solved, _ := ParsePrefixCommand(".solved")
	falseReport, _ := ParsePrefixCommand(".false")
	maybe, _ := ParsePrefixCommand(".maybe")
	tba, _ := ParsePrefixCommand(".tba")
	tbd, _ := ParsePrefixCommand(".tbd")

	if !PrefixActionAllowedForChannel(solved, IssuesChannelID) {
		t.Fatal("expected .solved to be allowed in issues")
	}
	if PrefixActionAllowedForChannel(solved, SuggestionChannelID) {
		t.Fatal("expected .solved to be rejected outside issues")
	}
	if !PrefixActionAllowedForChannel(falseReport, IssuesChannelID) {
		t.Fatal("expected .false to be allowed in issues")
	}
	if PrefixActionAllowedForChannel(falseReport, SuggestionChannelID) {
		t.Fatal("expected .false to be rejected outside issues")
	}
	if !PrefixActionAllowedForChannel(maybe, SuggestionChannelID) {
		t.Fatal("expected unrelated prefix action to remain allowed")
	}
	if !PrefixActionAllowedForChannel(tba, SuggestionChannelID) || !PrefixActionAllowedForChannel(tbd, SuggestionChannelID) {
		t.Fatal("expected .tba and .tbd to be allowed in suggestion")
	}
	if PrefixActionAllowedForChannel(tba, IssuesChannelID) || PrefixActionAllowedForChannel(tbd, IssuesChannelID) {
		t.Fatal("expected .tba and .tbd to be rejected outside suggestion")
	}
}
