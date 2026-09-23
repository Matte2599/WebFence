// Package i18n translates presentation text without changing domain values.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed en.json it.json
var files embed.FS

var catalogs = load()

func load() map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, locale := range []string{"en", "it"} {
		data, err := files.ReadFile(locale + ".json")
		if err != nil {
			panic(err) // Invalid embedded assets are a build defect, not user input.
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic(err)
		}
		result[locale] = catalog
	}
	return result
}

func Normalize(locale string) string {
	base := strings.FieldsFunc(strings.ToLower(locale), func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == '@'
	})
	if len(base) > 0 && base[0] == "it" {
		return "it"
	}
	return "en"
}

// Text falls back to English; unknown keys remain visible as defects.
func Text(locale, key string, args ...any) string {
	value := catalogs[Normalize(locale)][key]
	if value == "" {
		value = catalogs["en"][key]
	}
	if value == "" {
		return "[" + key + "]"
	}
	if len(args) == 0 {
		return value
	}
	return fmt.Sprintf(value, args...)
}
