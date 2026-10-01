package book

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/options"
	"github.com/tp86/legimi-go/internal/repository"
)

type cachedBook struct {
	Version uint64 `json:"version"`
	Title   string `json:"title"`
	Author  string `json:"author"`
}

// fileBookRepository remembers metadata of books seen on shelf. Legimi removes book from shelf
// listing once its download is requested, so remembered metadata allows to download it again.
type fileBookRepository struct {
	filePath string
}

// GetFileRepository stores books next to configuration file, e.g. config.ini -> config-books.json
func GetFileRepository(opts options.Configuration) repository.Book {
	configFile := opts.GetFile()
	return fileBookRepository{filePath: strings.TrimSuffix(configFile, filepath.Ext(configFile)) + "-books.json"}
}

func (r fileBookRepository) load() map[uint64]cachedBook {
	books := make(map[uint64]cachedBook)
	if data, err := os.ReadFile(r.filePath); err == nil {
		json.Unmarshal(data, &books)
	}
	return books
}

func (r fileBookRepository) Get(id uint64) (model.BookMetadata, bool) {
	book, ok := r.load()[id]
	return model.BookMetadata{Id: id, Version: book.Version, Title: book.Title, Author: book.Author}, ok
}

// Save adds or updates given books, keeping previously saved ones
func (r fileBookRepository) Save(books []model.BookMetadata) error {
	saved := r.load()
	for _, book := range books {
		saved[book.Id] = cachedBook{Version: book.Version, Title: book.Title, Author: book.Author}
	}
	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0600)
}
