// Package preferences stores only the desktop language, never project data.
package preferences

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func Load(path, fallback string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fallback, nil
	}
	if err != nil {
		return fallback, err
	}
	language := strings.TrimSpace(string(data))
	if language != "it" && language != "en" {
		return fallback, errors.New("invalid saved language")
	}
	return language, nil
}

func Save(path, language string) error {
	if language != "it" && language != "en" {
		return errors.New("invalid language")
	}
	if path == "" {
		return errors.New("no preference path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".language-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.WriteString(language + "\n"); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
