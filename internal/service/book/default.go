package book

import (
	"errors"
	"fmt"
	"time"

	"github.com/tp86/legimi-go/internal/api"
	"github.com/tp86/legimi-go/internal/api/protocol"
	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/service"
)

type defaultBookService struct {
	sessionService    service.Session
	client            api.Client
	downloadPresenter service.DownloadPresenter
}

func (bs defaultBookService) ListBooks() ([]model.BookMetadata, model.DownloadLimit, error) {
	// TODO better error handling
	session, err := bs.sessionService.GetSession()
	if err != nil {
		return nil, model.DownloadLimit{}, err
	}
	limit := session.DownloadLimit()
	list := make([]model.BookMetadata, 0)
	request := model.NewBookListRequest(session.Id)
	var bookList model.BookList
	for {
		err := bs.client.Exchange(request, &bookList)
		if err != nil {
			return list, limit, err
		}
		if len(bookList) == 0 {
			break
		}
		for _, book := range bookList {
			list = append(list, book)
		}
		request.NextPage = bookList[len(bookList)-1].NextPage
	}
	return list, limit, nil
}

func (bs defaultBookService) DownloadBooks(bookIds []uint64) error {
	// TODO concurrent download of all books
	errs := make([]error, 0)
	for _, id := range bookIds {
		errs = append(errs, bs.downloadBook(id))
	}
	return errors.Join(errs...)
}

func (bs defaultBookService) downloadBook(id uint64) error {
	// TODO concurrent downloader
	session, err := bs.sessionService.GetSession()
	if err != nil {
		return err
	}
	sessionId := session.Id
	book, err := bs.getBookMetadata(sessionId, id)
	if err != nil {
		return err
	}
	bookDownloadDetails, err := bs.getBookDownloadDetails(sessionId, book)
	if err != nil {
		return err
	}
	return bs.download(book, bookDownloadDetails)
}

func (bs defaultBookService) getBookMetadata(sessionId string, bookId uint64) (model.BookMetadata, error) {
	metadataRequest := model.NewBookListRequest(sessionId)
	metadataRequest.BookId = bookId
	var bookList model.BookList
	err := bs.client.Exchange(metadataRequest, &bookList)
	if err != nil {
		return model.BookMetadata{}, err
	}
	if len(bookList) != 1 {
		return model.BookMetadata{}, fmt.Errorf("unexpected book metadata list count: %d, expected 1", len(bookList))
	}
	return bookList[0], nil
}

const maxDownloadDetailsGetAttempts = 5

func (bs defaultBookService) getBookDownloadDetails(sessionId string, book model.BookMetadata) (model.BookDownloadDetails, error) {
	downloadDetailsRequest := model.NewBookDownloadDetailsRequest(sessionId, book.Id, book.Version)
	var bookDownloadDetails model.BookDownloadDetails
	// TODO refactor & test
	attempt := 0
	for ; attempt < maxDownloadDetailsGetAttempts; attempt++ {
		if err := bs.client.Exchange(downloadDetailsRequest, &bookDownloadDetails); err != nil {
			if err, ok := err.(protocol.ErrorResponse); ok && err.Type == protocol.BookDownloadDetailsPreparingError {
				// special case - download details are being prepared, try to repeat after some time
				bs.downloadPresenter.Wait(book)
				time.Sleep(2 * time.Second)
				continue
			}
			return bookDownloadDetails, err
		}
		break
	}
	if attempt == maxDownloadDetailsGetAttempts {
		return bookDownloadDetails, fmt.Errorf("couldn't get download details after %d attempts, try downloading book again after some time", attempt)
	}
	return bookDownloadDetails, nil
}
