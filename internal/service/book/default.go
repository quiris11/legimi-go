package book

import (
	"fmt"
	"os"
	"strings"
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
	// empty for current directory
	downloadDirectory string
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

func (bs defaultBookService) CheckDownloadDirectory() error {
	return checkDirectory(bs.downloadDirectory)
}

func (bs defaultBookService) DownloadBooks(bookIds []uint64) error {
	// TODO concurrent download of all books
	if err := bs.CheckDownloadDirectory(); err != nil {
		return err
	}
	// failure of one book doesn't stop downloading the others
	var failed []string
	for _, id := range bookIds {
		if book, err := bs.downloadBook(id); err != nil {
			bs.downloadPresenter.Skip(book, err)
			failed = append(failed, fmt.Sprint(id))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d of %d book(s) not downloaded: %s", len(failed), len(bookIds), strings.Join(failed, " "))
	}
	return nil
}

// downloadBook returns metadata of the book (at least its id) also when download fails
func (bs defaultBookService) downloadBook(id uint64) (model.BookMetadata, error) {
	// TODO concurrent downloader
	book := model.BookMetadata{Id: id}
	session, err := bs.sessionService.GetSession()
	if err != nil {
		return book, err
	}
	sessionId := session.Id
	book, err = bs.getBookMetadata(sessionId, id)
	if err != nil {
		return model.BookMetadata{Id: id}, err
	}
	// asking Legimi for download would hide the book from listing, although download would be refused
	if limit := session.DownloadLimit(); limit.IsKnown() && limit.Left == 0 && !book.Downloaded && book.LastDownloaded.IsZero() {
		return book, fmt.Errorf("no downloads left in this subscription period (Legimi was not asked, book stays on the list)")
	}
	// Legimi hides book from shelf listing from now on, remember it
	if err := bs.bookRepository.DownloadRequested(book); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't remember download of book %d: %v\n", id, err)
	}
	bookDownloadDetails, err := bs.getBookDownloadDetails(sessionId, book)
	if err != nil {
		return book, err
	}
	if err := bs.download(book, bookDownloadDetails); err != nil {
		return book, err
	}
	if err := bs.bookRepository.Downloaded(id); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't remember download of book %d: %v\n", id, err)
	}
	return book, nil
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
		book, ok := bs.bookRepository.Get(bookId)
		if !ok {
			book = model.BookMetadata{Id: bookId, Title: unknownTitle}
		}
		// remembered version may be outdated (new edition released since), always download current one
		book.Version, err = bs.findCurrentVersion(sessionId, bookId)
		return book, err
	}
	if len(bookList) != 1 {
		return model.BookMetadata{}, fmt.Errorf("unexpected book metadata list count: %d, expected 1", len(bookList))
	}
	return bookList[0], nil
}

const maxBookVersion = 50

// findCurrentVersion asks for consecutive versions of the book until Legimi reports that version doesn't exist.
// Older version may not be available anymore (it is being prepared forever), so the current one has to be used.
func (bs defaultBookService) findCurrentVersion(sessionId string, bookId uint64) (uint64, error) {
	var current uint64
	for version := uint64(1); version <= maxBookVersion; version++ {
		var details model.BookDownloadDetails
		err := bs.client.Exchange(model.NewBookDownloadDetailsRequest(sessionId, bookId, version), &details)
		if err, ok := err.(protocol.ErrorResponse); ok && err.Type == protocol.BookVersionNotAvailableError {
			break
		}
		if err != nil {
			if err, ok := err.(protocol.ErrorResponse); !ok || err.Type != protocol.BookDownloadDetailsPreparingError {
				return 0, err
			}
		}
		current = version
	}
	if current == 0 {
		return 0, fmt.Errorf("book %d is not available for download", bookId)
	}
	return current, nil
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
