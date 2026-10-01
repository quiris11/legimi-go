package book

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/options"
	"github.com/tp86/legimi-go/internal/repository"
)

type cachedBook struct {
	Version    uint64 `json:"version"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Downloaded bool   `json:"downloaded"` // as last listed by Legimi
	// downloads made with this program
	DownloadRequested *time.Time `json:"downloadRequested,omitempty"`
	LastDownloaded    *time.Time `json:"lastDownloaded,omitempty"`
}

func (b cachedBook) metadata(id uint64) model.BookMetadata {
	book := model.BookMetadata{
		Id:         id,
		Version:    b.Version,
		Title:      b.Title,
		Author:     b.Author,
		Downloaded: b.Downloaded || b.LastDownloaded != nil,
	}
	if b.LastDownloaded != nil {
		book.LastDownloaded = *b.LastDownloaded
	}
	return book
}

// fileBookRepository remembers metadata of books seen on shelf and downloads made with this program.
// Legimi hides book from shelf listing once its download is requested, so remembered metadata
// allows to show it and download it again.
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

func (r fileBookRepository) store(books map[uint64]cachedBook) error {
	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0600)
}

func (r fileBookRepository) Get(id uint64) (model.BookMetadata, bool) {
	book, ok := r.load()[id]
	return book.metadata(id), ok
}

func (r fileBookRepository) Save(books []model.BookMetadata) error {
	saved := r.load()
	for _, book := range books {
		cached := saved[book.Id]
		cached.Version, cached.Title, cached.Author, cached.Downloaded = book.Version, book.Title, book.Author, book.Downloaded
		saved[book.Id] = cached
	}
	return r.store(saved)
}

func (r fileBookRepository) DownloadRequested(book model.BookMetadata) error {
	saved := r.load()
	cached, known := saved[book.Id]
	if !known {
		cached = cachedBook{Version: book.Version, Title: book.Title, Author: book.Author, Downloaded: book.Downloaded}
	}
	if book.Version != 0 {
		// e.g. current version found for book hidden by Legimi
		cached.Version = book.Version
	}
	now := time.Now()
	cached.DownloadRequested = &now
	saved[book.Id] = cached
	return r.store(saved)
}

func (r fileBookRepository) Downloaded(id uint64) error {
	saved := r.load()
	cached := saved[id]
	now := time.Now()
	cached.LastDownloaded = &now
	saved[id] = cached
	return r.store(saved)
}

func (r fileBookRepository) Complete(listed []model.BookMetadata) []model.BookMetadata {
	saved := r.load()
	books := make([]model.BookMetadata, 0, len(listed))
	isListed := make(map[uint64]bool)
	for _, book := range listed {
		if cached, ok := saved[book.Id]; ok && cached.LastDownloaded != nil {
			book.LastDownloaded = *cached.LastDownloaded
		}
		books = append(books, book)
		isListed[book.Id] = true
	}
	for id, cached := range saved {
		if cached.DownloadRequested != nil && !isListed[id] {
			book := cached.metadata(id)
			book.Hidden = true
			books = append(books, book)
		}
	}
	return books
}
