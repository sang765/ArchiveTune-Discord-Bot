package discord

import "testing"

func TestParseRejectCommand(t *testing.T) {
	reason, matched, valid := ParseRejectCommand(".reject This suggestion is outside the project scope")
	if !matched || !valid || reason != "This suggestion is outside the project scope" {
		t.Fatalf("unexpected reject parse: reason=%q matched=%v valid=%v", reason, matched, valid)
	}
}

func TestParseRejectCommandRequiresReason(t *testing.T) {
	if _, matched, valid := ParseRejectCommand(".reject"); !matched || valid {
		t.Fatal("expected missing reason to be matched but invalid")
	}
}

func TestParseRejectCommandRejectsLongReason(t *testing.T) {
	longReason := ".reject " + string(make([]byte, 1001))
	if _, matched, valid := ParseRejectCommand(longReason); !matched || valid {
		t.Fatal("expected long reason to be rejected")
	}
}
