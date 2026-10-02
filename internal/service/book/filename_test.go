package book

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tp86/legimi-go/internal/model"
)

func TestBookFileName(t *testing.T) {
	tests := map[string]string{
		"Lalka":                  "Lalka (1000001).mobi",
		"Nad Niemnem":            "Nad Niemnem (1000001).mobi",
		"Kto? Co? Gdzie?":        "Kto Co Gdzie (1000001).mobi",
		"Wojna: historia":        "Wojna - historia (1000001).mobi",
		`"Cytat" a/b\c|d`:        "'Cytat' a-b-c-d (1000001).mobi",
		"Gwiazdka* <i> nawiasy>": "Gwiazdka i nawiasy (1000001).mobi",
		"Koniec zdania...":       "Koniec zdania (1000001).mobi",
		"  Za   dużo\tspacji  ":  "Za dużo spacji (1000001).mobi",
		"Zażółć gęślą jaźń":      "Zażółć gęślą jaźń (1000001).mobi",
		"":                       "1000001.mobi",
		"???":                    "1000001.mobi",
		unknownTitle:             "1000001.mobi",
	}
	for title, expected := range tests {
		if got := bookFileName(model.BookMetadata{Id: 1000001, Title: title}); got != expected {
			t.Errorf("%q: got %q, expected %q", title, got, expected)
		}
	}
}

func TestLongTitleIsShortenedWithoutBreakingCharacters(t *testing.T) {
	name := bookFileName(model.BookMetadata{Id: 1000001, Title: strings.Repeat("ż", 300)})
	if !utf8.ValidString(name) || len(name+".part") > 255 || !strings.HasSuffix(name, " (1000001).mobi") {
		t.Errorf("%d bytes: %q", len(name), name)
	}
}
