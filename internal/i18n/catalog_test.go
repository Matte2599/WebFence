package i18n

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestCatalogsCompleteAndFormattingCompatible(t *testing.T) {
	format := regexp.MustCompile(`%[ds]`)
	for locale, catalog := range catalogs {
		if len(catalog) != len(catalogs["en"]) {
			t.Fatalf("%s has different key count", locale)
		}
		for key, reference := range catalogs["en"] {
			translation := catalog[key]
			if strings.TrimSpace(translation) == "" {
				t.Errorf("%s missing %s", locale, key)
			}
			if !slices.Equal(format.FindAllString(reference, -1), format.FindAllString(translation, -1)) {
				t.Errorf("%s incompatible placeholders in %s", locale, key)
			}
		}
		if got := Text(locale, "count", 0, 10000); strings.Contains(got, "%!") {
			t.Errorf("bad count: %s", got)
		}
	}
}

func TestLocaleFallback(t *testing.T) {
	for input, want := range map[string]string{"it-IT": "it", "IT_it.UTF-8": "it", "it": "it", "en-US": "en", "fr": "en", "": "en", "C": "en"} {
		if got := Normalize(input); got != want {
			t.Errorf("%q: got %q, want %q", input, got, want)
		}
	}
	if Text("fr", "copy") != Text("en", "copy") {
		t.Fatal("English fallback failed")
	}
	if Text("it", "missing") != "[missing]" {
		t.Fatal("unknown key hidden")
	}
}
