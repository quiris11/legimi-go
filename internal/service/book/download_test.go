package book

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tp86/legimi-go/internal/model"
)

func init() {
	chunkRetryDelay = 0
}

// makeMobi builds minimal valid Mobipocket file with given number of records of given size
func makeMobi(records, recordSize int) []byte {
	header := make([]byte, mobiRecordListStart+(records+1)*mobiRecordInfoSize+2)
	copy(header[mobiTypeOffset:], mobiType)
	binary.BigEndian.PutUint16(header[mobiRecordCountAt:], uint16(records+1))
	offset := len(header)
	for i := 0; i <= records; i++ {
		binary.BigEndian.PutUint32(header[mobiRecordListStart+i*mobiRecordInfoSize:], uint32(offset))
		offset += recordSize
	}
	data := header
	for i := 0; i < records; i++ {
		data = append(data, bytes.Repeat([]byte{byte(i)}, recordSize)...)
	}
	return append(data, mobiEndOfFile...)
}

func TestValidMobiPassesVerification(t *testing.T) {
	book := makeMobi(10, 1000)
	if err := verifyMobiData(book, uint64(len(book))); err != nil {
		t.Error(err)
	}
}

func TestCorruptedMobiFailsVerification(t *testing.T) {
	book := makeMobi(10, 1000)
	html := []byte("<html><body>Access denied</body></html>")
	tests := map[string]struct {
		data []byte
		size int
	}{
		"truncated":      {book[:len(book)-100], len(book) - 100},
		"too short":      {book[:len(book)-100], len(book)},
		"too long":       {append(append([]byte{}, book...), 0), len(book)},
		"html":           {html, len(html)},
		"no end marker":  {append(append([]byte{}, book[:len(book)-4]...), 0, 0, 0, 0), len(book)},
		"part duplicate": {append(append([]byte{}, book[:2000]...), book[1000:len(book)-1000]...), len(book)},
	}
	for name, test := range tests {
		if err := verifyMobiData(test.data, uint64(test.size)); err == nil {
			t.Errorf("%s: corrupted book passed verification", name)
		}
	}
}

// server serves book supporting Range requests; misbehave can alter response for n-th request
func server(t *testing.T, book []byte, misbehave func(n int, w http.ResponseWriter, first, last int) bool) *httptest.Server {
	n := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		var first, last int
		if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &first, &last); err != nil {
			t.Errorf("invalid Range header: %q", r.Header.Get("Range"))
		}
		if misbehave != nil && misbehave(n, w, first, last) {
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, last, len(book)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(book[first : last+1])
	}))
}

func downloadFromServer(t *testing.T, book []byte, misbehave func(int, http.ResponseWriter, int, int) bool) ([]byte, error) {
	srv := server(t, book, misbehave)
	defer srv.Close()
	fileName := filepath.Join(t.TempDir(), "book.mobi")
	err := downloadToFile(fileName, model.BookDownloadDetails{Url: srv.URL, Size: uint64(len(book))}, func() {})
	if err == nil {
		err = verifyMobi(fileName, uint64(len(book)))
	}
	data, _ := os.ReadFile(fileName)
	return data, err
}

func TestDownloadInChunks(t *testing.T) {
	book := makeMobi(30, 10000)
	data, err := downloadFromServer(t, book, nil)
	if err != nil || !bytes.Equal(data, book) {
		t.Errorf("downloaded book differs from original, error: %v", err)
	}
}

func TestDownloadResumesAfterBrokenConnection(t *testing.T) {
	book := makeMobi(30, 10000)
	data, err := downloadFromServer(t, book, func(n int, w http.ResponseWriter, first, last int) bool {
		if n == 2 {
			// send only part of the chunk, then break connection
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, last, len(book)))
			w.Header().Set("Content-Length", fmt.Sprint(last-first+1))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(book[first : first+1000])
			panic(http.ErrAbortHandler)
		}
		return false
	})
	if err != nil || !bytes.Equal(data, book) {
		t.Errorf("downloaded book differs from original, error: %v", err)
	}
}

func TestDownloadWhenServerIgnoresRange(t *testing.T) {
	book := makeMobi(30, 10000)
	data, err := downloadFromServer(t, book, func(n int, w http.ResponseWriter, first, last int) bool {
		w.WriteHeader(http.StatusOK)
		w.Write(book)
		return true
	})
	if err != nil || !bytes.Equal(data, book) {
		t.Errorf("downloaded book differs from original, error: %v", err)
	}
}

func TestDownloadFailsOnServerMisbehavior(t *testing.T) {
	book := makeMobi(30, 10000)
	tests := map[string]func(int, http.ResponseWriter, int, int) bool{
		"error status": func(n int, w http.ResponseWriter, first, last int) bool {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("<html>Forbidden</html>"))
			return true
		},
		"whole book in the middle": func(n int, w http.ResponseWriter, first, last int) bool {
			if first == 0 {
				return false
			}
			w.WriteHeader(http.StatusOK)
			w.Write(book)
			return true
		},
		"wrong part": func(n int, w http.ResponseWriter, first, last int) bool {
			if first == 0 {
				return false
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first-100, last-100, len(book)))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(book[first-100 : last-99])
			return true
		},
		"no data": func(n int, w http.ResponseWriter, first, last int) bool {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, last, len(book)))
			w.WriteHeader(http.StatusPartialContent)
			return true
		},
	}
	for name, misbehave := range tests {
		if _, err := downloadFromServer(t, book, misbehave); err == nil {
			t.Errorf("%s: download should fail", name)
		}
	}
}

type stubPresenter struct{ failed, ended bool }

func (p *stubPresenter) Start(model.BookMetadata) {}
func (p *stubPresenter) Part(model.BookMetadata)  {}
func (p *stubPresenter) End(model.BookMetadata)   { p.ended = true }
func (p *stubPresenter) Fail(model.BookMetadata)  { p.failed = true }
func (p *stubPresenter) Wait(model.BookMetadata)  {}

func TestFailedDownloadKeepsExistingBookFile(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)
	existing := []byte("previously downloaded book")
	os.WriteFile("123.mobi", existing, 0644)

	book := makeMobi(30, 10000)
	srv := server(t, book, func(n int, w http.ResponseWriter, first, last int) bool {
		if first == 0 {
			return false
		}
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	defer srv.Close()
	presenter := &stubPresenter{}
	bs := defaultBookService{downloadPresenter: presenter}
	err := bs.download(model.BookMetadata{Id: 123}, model.BookDownloadDetails{Url: srv.URL, Size: uint64(len(book))})

	if err == nil || !strings.Contains(err.Error(), "book 123") || !presenter.failed {
		t.Errorf("download should fail, error: %v", err)
	}
	if data, _ := os.ReadFile("123.mobi"); !bytes.Equal(data, existing) {
		t.Error("existing book file was changed")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("temporary file left: %v", entries)
	}
}

func TestSuccessfulDownloadReplacesBookFile(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)
	os.WriteFile("123.mobi", []byte("old"), 0644)

	book := makeMobi(30, 10000)
	srv := server(t, book, nil)
	defer srv.Close()
	presenter := &stubPresenter{}
	bs := defaultBookService{downloadPresenter: presenter}
	if err := bs.download(model.BookMetadata{Id: 123}, model.BookDownloadDetails{Url: srv.URL, Size: uint64(len(book))}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile("123.mobi"); !bytes.Equal(data, book) || !presenter.ended {
		t.Error("book file not replaced with downloaded book")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("temporary file left: %v", entries)
	}
}
