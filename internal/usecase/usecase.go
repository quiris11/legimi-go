package usecase

import "github.com/tp86/legimi-go/internal/model"

type BookLister interface {
	ListBooks() ([]model.BookMetadata, model.DownloadLimit, error)
}

type DeviceRefresher interface {
	RefreshDevice() (uint64, error)
}

type BookDownloader interface {
	DownloadBooks([]uint64) error
	// CheckDownloadDirectory makes sure books can be downloaded, e.g. that Kindle is mounted
	CheckDownloadDirectory() error
}
