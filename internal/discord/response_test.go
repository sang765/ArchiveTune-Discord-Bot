package discord

import "testing"

func TestResponseEmbeds(t *testing.T) {
	cases := []struct {
		name       string
		embedTitle string
		color      int
	}{
		{name: "info", embedTitle: InfoEmbed("Usage", "details").Title, color: InfoEmbed("Usage", "details").Color},
		{name: "success", embedTitle: SuccessEmbed("Done", "details").Title, color: SuccessEmbed("Done", "details").Color},
		{name: "error", embedTitle: ErrorEmbed("Failed", "details").Title, color: ErrorEmbed("Failed", "details").Color},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.embedTitle == "" || testCase.embedTitle[:len("ArchiveTune Bot")] != "ArchiveTune Bot" {
				t.Fatalf("expected ArchiveTune Bot title, got %q", testCase.embedTitle)
			}
			if testCase.color == 0 {
				t.Fatal("expected response color")
			}
		})
	}
}
