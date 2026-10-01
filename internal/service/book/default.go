package book

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tp86/legimi-go/internal/api"
	"github.com/tp86/legimi-go/internal/api/protocol"
	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/repository"
	"github.com/tp86/legimi-go/internal/service"
)

type defaultBookService struct {
	sessionService    service.Session
	client            api.Client
	downloadPresenter service.DownloadPresenter
	bookRepository    repository.Book
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
	if err := bs.bookRepository.Save(list); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't remember books metadata: %v\n", err)
	}
	return bs.bookRepository.Complete(list), limit, nil
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
	// Legimi hides book from shelf listing from now on, remember it
	if err := bs.bookRepository.DownloadRequested(book); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't remember download of book %d: %v\n", id, err)
	}
	bookDownloadDetails, err := bs.getBookDownloadDetails(sessionId, book)
	if err != nil {
		return err
	}
	if err := bs.download(book, bookDownloadDetails); err != nil {
		return err
	}
	if err := bs.bookRepository.Downloaded(id); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't remember download of book %d: %v\n", id, err)
	}
	return nil
}

func (bs defaultBookService) getBookMetadata(sessionId string, bookId uint64) (model.BookMetadata, error) {
	metadataRequest := model.NewBookListRequest(sessionId)
	metadataRequest.BookId = bookId
	var bookList model.BookList
	err := bs.client.Exchange(metadataRequest, &bookList)
	if err != nil {
		return model.BookMetadata{}, err
	}
	if len(bookList) == 0 {
		// Legimi removes book from shelf listing once its download is requested
		if book, ok := bs.bookRepository.Get(bookId); ok {
			return book, nil
		}
		// version not lower than current one is accepted and current book file is sent
		return model.BookMetadata{Id: bookId, Version: 1, Title: "(title unknown)"}, nil
	}
	if len(bookList) != 1 {
		return model.BookMetadata{}, fmt.Errorf("unexpected book metadata list count: %d, expected 1", len(bookList))
	}
	return bookList[0], nil
}

// preparing book file by Legimi may take a while, wait with increasing delays
var downloadDetailsDelays = []time.Duration{2, 3, 5, 10, 15, 20, 30, 30, 30, 30}

const downloadDetailsDelayUnit = time.Second

func (bs defaultBookService) getBookDownloadDetails(sessionId string, book model.BookMetadata) (model.BookDownloadDetails, error) {
	downloadDetailsRequest := model.NewBookDownloadDetailsRequest(sessionId, book.Id, book.Version)
	var bookDownloadDetails model.BookDownloadDetails
	// TODO refactor & test
	for _, delay := range downloadDetailsDelays {
		err := bs.client.Exchange(downloadDetailsRequest, &bookDownloadDetails)
		if err == nil {
			return bookDownloadDetails, nil
		}
		if err, ok := err.(protocol.ErrorResponse); !ok || err.Type != protocol.BookDownloadDetailsPreparingError {
			return bookDownloadDetails, err
		}
		// special case - download details are being prepared, try to repeat after some time
		bs.downloadPresenter.Wait(book)
		time.Sleep(delay * downloadDetailsDelayUnit)
	}
	return bookDownloadDetails, fmt.Errorf("book %d is still being prepared by Legimi, try downloading it again later with: download %d", book.Id, book.Id)
}
