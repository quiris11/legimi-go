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
	if !ok || book.Version != 2 || book.Title != "Pan Tadeusz" || book.Author != "Adam Mickiewicz" {
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

func TestDownloadedBooksHiddenByLegimiAreAddedToList(t *testing.T) {
	repository := GetFileRepository(configFile(filepath.Join(t.TempDir(), "config.ini")))
	listed := []model.BookMetadata{
		{Id: 1, Version: 1, Title: "Listed", Author: "A"},
		{Id: 2, Version: 1, Title: "Downloaded", Author: "B", Downloaded: true},
		{Id: 3, Version: 1, Title: "Requested", Author: "C"},
		{Id: 4, Version: 1, Title: "Removed from shelf", Author: "D"},
	}
	repository.Save(listed)
	book, _ := repository.Get(2)
	repository.DownloadRequested(book)
	repository.Downloaded(2)
	book, _ = repository.Get(3)
	repository.DownloadRequested(book)
	// download of book 1 is requested, but it's still listed
	book, _ = repository.Get(1)
	repository.DownloadRequested(book)
	repository.Downloaded(1)

	// Legimi hides books 2 and 3 after download request, book 4 was removed from shelf
	books := repository.Complete(listed[:1])
	byId := make(map[uint64]model.BookMetadata)
	for _, book := range books {
		byId[book.Id] = book
	}
	if len(books) != 3 {
		t.Fatalf("got %+v", books)
	}
	if b := byId[1]; b.Hidden || b.LastDownloaded.IsZero() {
		t.Errorf("listed book: %+v", b)
	}
	if b := byId[2]; !b.Hidden || !b.Downloaded || b.LastDownloaded.IsZero() || b.Title != "Downloaded" {
		t.Errorf("hidden downloaded book: %+v", b)
	}
	if b := byId[3]; !b.Hidden || b.Downloaded || !b.LastDownloaded.IsZero() {
		t.Errorf("hidden requested book: %+v", b)
	}
	if _, ok := byId[4]; ok {
		t.Error("book removed from shelf without download request should not be listed")
	}
}
