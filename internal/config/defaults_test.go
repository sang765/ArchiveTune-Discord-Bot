package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEnsureDefaultsAddsMissingYTDSetting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := "bot_token: token\nguild_id: guild\n# Keep this comment.\nprefix_max_distance: 3\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	changed, err := EnsureDefaults(path)
	if err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	if !changed {
		t.Fatal("EnsureDefaults() changed = false, want true")
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}
	if !strings.Contains(string(updated), "# Keep this comment.") {
		t.Fatalf("existing comment was not preserved: %s", updated)
	}

	var document map[string]any
	if err := yaml.Unmarshal(updated, &document); err != nil {
		t.Fatalf("parse updated config: %v", err)
	}
	ytd, ok := document["ytd"].(map[string]any)
	if !ok {
		t.Fatalf("ytd config = %#v, want mapping", document["ytd"])
	}
	if got, ok := ytd["block_playlist_album_download"].(bool); !ok || !got {
		t.Fatalf("block_playlist_album_download = %#v, want true", ytd["block_playlist_album_download"])
	}
	if got, ok := document["prefix_max_distance"].(int); !ok || got != 3 {
		t.Fatalf("prefix_max_distance = %#v, want 3", document["prefix_max_distance"])
	}
}

func TestEnsureDefaultsDoesNotOverwriteExistingYTDSetting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := "bot_token: token\nguild_id: guild\nytd:\n  block_playlist_album_download: false\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	changed, err := EnsureDefaults(path)
	if err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	if changed {
		t.Fatal("EnsureDefaults() changed = true, want false")
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(updated) != original {
		t.Fatalf("existing config was rewritten:\noriginal=%q\nupdated=%q", original, string(updated))
	}
}

func TestEnsureDefaultsIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("bot_token: token\nguild_id: guild\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	changed, err := EnsureDefaults(path)
	if err != nil || !changed {
		t.Fatalf("first EnsureDefaults() = changed %v, err %v; want changed true", changed, err)
	}
	changed, err = EnsureDefaults(path)
	if err != nil {
		t.Fatalf("second EnsureDefaults() error = %v", err)
	}
	if changed {
		t.Fatal("second EnsureDefaults() changed = true, want false")
	}
}
