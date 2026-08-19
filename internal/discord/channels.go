package discord

const (
	IssuesChannelID     = "1498327801923637439"
	SuggestionChannelID = "1498328044635422790"
)

func PrefixActionAllowedForChannel(action PrefixAction, parentChannelID string) bool {
	switch action.Command {
	case ".solved", ".false", ".false-report":
		return parentChannelID == IssuesChannelID
	default:
		return true
	}
}
