package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := map[string]string{
		"~":                  home,
		"~/Kindle/documents": filepath.Join(home, "Kindle/documents"),
		"/media/Kindle":      "/media/Kindle",
		"books":              "books",
		"~user/books":        "~user/books",
		"":                   "",
	}
	for path, expected := range tests {
		if got := ExpandHome(path); got != expected {
			t.Errorf("%q: got %q, expected %q", path, got, expected)
		}
	}
}
