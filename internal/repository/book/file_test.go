package book

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tp86/legimi-go/internal/model"
)

type configFile string

func (c configFile) GetFile() string { return string(c) }

func TestBooksAreRememberedAcrossSaves(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.ini")
	repository := GetFileRepository(configFile(config))
	if _, ok := repository.Get(1); ok {
		t.Error("book found in empty repository")
	}
	repository.Save([]model.BookMetadata{{Id: 1, Version: 2, Title: "Pan Tadeusz", Author: "Adam Mickiewicz"}})
	// book removed from shelf listing is kept
	repository.Save([]model.BookMetadata{{Id: 3, Version: 1, Title: "Lalka", Author: "Bolesław Prus"}})

	book, ok := GetFileRepository(configFile(config)).Get(1)
	if !ok || book != (model.BookMetadata{Id: 1, Version: 2, Title: "Pan Tadeusz", Author: "Adam Mickiewicz"}) {
		t.Errorf("got %+v, found %v", book, ok)
	}
	if _, ok := repository.Get(3); !ok {
		t.Error("book 3 not found")
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(config), "config-books.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Errorf("books file: %v, %v", info, err)
	}
}
