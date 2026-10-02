package book

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tp86/legimi-go/internal/api/protocol"
	"github.com/tp86/legimi-go/internal/model"
	bookrepository "github.com/tp86/legimi-go/internal/repository/book"
)

type sessionStub struct{ limit model.DownloadLimit }

func (s sessionStub) GetSession() (model.Session, error) {
	return model.NewSession("session", s.limit), nil
}

type configFile string

func (c configFile) GetFile() string { return string(c) }

// shelfClient lists books of the shelf and gives download details, refusing download of books in refused
type shelfClient struct {
	shelf            map[uint64]model.BookMetadata
	refused          map[uint64]bool
	url              string
	size             uint64
	detailsRequested []uint64
}

func (c *shelfClient) Exchange(request protocol.Request, response protocol.Response) error {
	switch response := response.(type) {
	case *model.BookList:
		if book, ok := c.shelf[request.(model.BookListRequest).BookId]; ok {
			*response = model.BookList{book}
		}
	case *model.BookDownloadDetails:
		var buf bytes.Buffer
		request.Encode(&buf)
		// request starts with book id
		id := binary.LittleEndian.Uint64(buf.Bytes()[:8])
		c.detailsRequested = append(c.detailsRequested, id)
		if c.refused[id] {
			return protocol.ErrorResponse{Type: protocol.DownloadRefusedError}
		}
		*response = model.BookDownloadDetails{Url: c.url, Size: c.size}
	}
	return nil
}

func newShelfService(t *testing.T, limit model.DownloadLimit, client *shelfClient) (defaultBookService, *stubPresenter, string) {
	directory := t.TempDir()
	presenter := &stubPresenter{}
	return defaultBookService{
		sessionService:    sessionStub{limit},
		client:            client,
		downloadPresenter: presenter,
		bookRepository:    bookrepository.GetFileRepository(configFile(filepath.Join(t.TempDir(), "config.ini"))),
		downloadDirectory: directory,
	}, presenter, directory
}

func shelf(books ...model.BookMetadata) map[uint64]model.BookMetadata {
	m := make(map[uint64]model.BookMetadata)
	for _, book := range books {
		m[book.Id] = book
	}
	return m
}

func downloadedFiles(directory string) []string {
	entries, _ := os.ReadDir(directory)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestRefusedBookIsSkippedAndOthersAreDownloaded(t *testing.T) {
	book := makeMobi(30, 10000)
	srv := server(t, book, nil)
	defer srv.Close()
	client := &shelfClient{
		shelf:   shelf(model.BookMetadata{Id: 1, Version: 1}, model.BookMetadata{Id: 2, Version: 1}, model.BookMetadata{Id: 3, Version: 1}),
		refused: map[uint64]bool{2: true},
		url:     srv.URL,
		size:    uint64(len(book)),
	}
	bs, presenter, directory := newShelfService(t, model.DownloadLimit{Left: 5, Max: 10}, client)

	err := bs.DownloadBooks([]uint64{1, 2, 3})

	if err == nil || !strings.Contains(err.Error(), "1 of 3 book(s) not downloaded: 2") {
		t.Errorf("error: %v", err)
	}
	if !slices.Equal(presenter.skipped, []uint64{2}) {
		t.Errorf("skipped %v", presenter.skipped)
	}
	if files := downloadedFiles(directory); !slices.Equal(files, []string{"1.mobi", "3.mobi"}) {
		t.Errorf("downloaded %v", files)
	}
}

func TestWithoutDownloadsLeftOnlyDownloadedBooksAreRequested(t *testing.T) {
	book := makeMobi(30, 10000)
	srv := server(t, book, nil)
	defer srv.Close()
	client := &shelfClient{
		shelf: shelf(
			model.BookMetadata{Id: 1, Version: 1, Downloaded: true},
			model.BookMetadata{Id: 2, Version: 1},
		),
		url:  srv.URL,
		size: uint64(len(book)),
	}
	bs, presenter, directory := newShelfService(t, model.DownloadLimit{Left: 0, Max: 10}, client)

	err := bs.DownloadBooks([]uint64{2, 1})

	if err == nil || !strings.Contains(err.Error(), "1 of 2 book(s) not downloaded: 2") {
		t.Errorf("error: %v", err)
	}
	// Legimi is not asked for book 2, so it's not hidden from listing
	if !slices.Equal(client.detailsRequested, []uint64{1}) || !slices.Equal(presenter.skipped, []uint64{2}) {
		t.Errorf("details requested %v, skipped %v", client.detailsRequested, presenter.skipped)
	}
	if files := downloadedFiles(directory); !slices.Equal(files, []string{"1.mobi"}) {
		t.Errorf("downloaded %v", files)
	}
}
