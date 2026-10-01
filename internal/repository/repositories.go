package repository

import "github.com/tp86/legimi-go/internal/model"

type Book interface {
	Get(id uint64) (model.BookMetadata, bool)
	// Save adds or updates metadata of listed books, keeping previously saved ones
	Save(books []model.BookMetadata) error
	DownloadRequested(book model.BookMetadata) error
	Downloaded(id uint64) error
	// Complete adds local information to listed books and appends books hidden by Legimi
	Complete(listed []model.BookMetadata) []model.BookMetadata
}

type Account interface {
	GetLogin() string
	GetPassword() string
	GetKindleId() uint64
	SaveLogin(login string)
	SavePassword(password string) error
	SaveKindleId(kindleId uint64)
}
