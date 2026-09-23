package preferences

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageRoundtripAndInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "ui-language")
	if value, err := Load(path, "it"); err != nil || value != "it" {
		t.Fatalf("missing preference: %q %v", value, err)
	}
	for _, language := range []string{"en", "it", "en"} {
		if err := Save(path, language); err != nil {
			t.Fatal(err)
		}
		if value, err := Load(path, "it"); err != nil || value != language {
			t.Fatalf("roundtrip: %q %v", value, err)
		}
	}
	if err := Save(path, "fr"); err == nil {
		t.Fatal("invalid language accepted")
	}
	if value, _ := Load(path, "it"); value != "en" {
		t.Fatal("invalid save changed preference")
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if value, err := Load(path, "it"); value != "it" || err == nil {
		t.Fatal("corruption not surfaced")
	}
}

func TestPreferenceIOErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir, "en"); err == nil {
		t.Fatal("read failure hidden")
	}
	if err := Save(dir, "it"); err == nil {
		t.Fatal("write failure hidden")
	}
	if err := Save("", "it"); err == nil {
		t.Fatal("missing path accepted")
	}
}
