package media

import "testing"

func TestSanitizeFileStem(t *testing.T) {
	got := sanitizeFileStem("Trót Tin Vào Lời Hứa - Hạ Vũ / Remix?")
	want := "Trót_Tin_Vào_Lời_Hứa_Hạ_Vũ_Remix"
	if got != want {
		t.Fatalf("sanitizeFileStem() = %q, want %q", got, want)
	}
}

func TestSanitizeFileStemFallback(t *testing.T) {
	if got := sanitizeFileStem("///"); got != "download" {
		t.Fatalf("sanitizeFileStem() = %q, want download", got)
	}
}
