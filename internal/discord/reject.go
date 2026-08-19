package discord

import "strings"

func ParseRejectCommand(content string) (reason string, matched, valid bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) == 0 || !strings.EqualFold(fields[0], ".reject") {
		return "", false, false
	}
	if len(fields) < 2 {
		return "", true, false
	}
	reason = strings.TrimSpace(strings.Join(fields[1:], " "))
	if reason == "" || len([]rune(reason)) > 1000 {
		return "", true, false
	}
	return reason, true, true
}
