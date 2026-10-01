package book

import (
	"github.com/tp86/legimi-go/internal/api"
	"github.com/tp86/legimi-go/internal/paths"
	"github.com/tp86/legimi-go/internal/repository"
	"github.com/tp86/legimi-go/internal/service"
)

func DefaultService(
	sessionService service.Session,
	apiClient api.Client,
	bookDownloadPresenter service.DownloadPresenter,
	bookRepository repository.Book,
	downloadDirectory string,
) service.Book {
	return defaultBookService{
		downloadDirectory: paths.ExpandHome(downloadDirectory),
		bookRepository:    bookRepository,
		sessionService:    sessionService,
		client:            apiClient,
		downloadPresenter: bookDownloadPresenter,
	}
}
