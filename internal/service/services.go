package service

import (
	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/usecase"
)

type Session interface {
	GetSession() (string, error)
}

type Account interface {
	GetCredentials() (string, string)
	// SaveCredentials stores credentials entered by user, should be called after successful login
	SaveCredentials()
	GetKindleId() (uint64, error)
}

type Book interface {
	usecase.BookLister
	usecase.BookDownloader
}

type BookListPresenter interface {
	Present([]model.BookMetadata)
}

type DownloadPresenter interface {
	Start(model.BookMetadata)
	Part(model.BookMetadata)
	End(model.BookMetadata)
	Wait(model.BookMetadata)
}
