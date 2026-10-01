package book

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/tp86/legimi-go/internal/model"
)

const (
	downloadChunkSize uint64 = 81920
	// attempts to download single chunk, e.g. when connection breaks
	maxChunkAttempts = 3
)

var chunkRetryDelay = 2 * time.Second

func (bs defaultBookService) download(book model.BookMetadata, downloadDetails model.BookDownloadDetails) error {
	if downloadDetails.Size == 0 {
		return fmt.Errorf("book %d: download size not received", book.Id)
	}
	bs.downloadPresenter.Start(book)
	fileName := fmt.Sprintf("%d.mobi", book.Id)
	// download to temporary file first, so that book file (possibly existing one)
	// is replaced only with complete and verified download
	partFileName := fileName + ".part"
	err := downloadToFile(partFileName, downloadDetails, func() { bs.downloadPresenter.Part(book) })
	if err == nil {
		err = verifyMobi(partFileName, downloadDetails.Size)
	}
	if err == nil {
		err = os.Rename(partFileName, fileName)
	}
	if err != nil {
		os.Remove(partFileName)
		bs.downloadPresenter.Fail(book)
		return fmt.Errorf("book %d: %v", book.Id, err)
	}
	bs.downloadPresenter.End(book)
	return nil
}

func downloadToFile(fileName string, downloadDetails model.BookDownloadDetails, chunkDone func()) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	err = downloadTo(file, downloadDetails, chunkDone)
	if err == nil {
		// make sure data is written to disk (e.g. Kindle connected via USB) before file is used
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func downloadTo(file *os.File, downloadDetails model.BookDownloadDetails, chunkDone func()) error {
	client := &http.Client{Timeout: 30 * time.Second}
	var downloadedBytes uint64
	failedAttempts := 0
	for downloadedBytes < downloadDetails.Size {
		// next chunk always starts where downloaded data ends, also after partial chunk
		written, err := downloadChunk(client, file, downloadDetails, downloadedBytes)
		downloadedBytes += written
		if err != nil {
			failedAttempts++
			if failedAttempts == maxChunkAttempts {
				return fmt.Errorf("download failed at %d of %d bytes: %v", downloadedBytes, downloadDetails.Size, err)
			}
			time.Sleep(chunkRetryDelay)
			continue
		}
		failedAttempts = 0
		chunkDone()
	}
	return nil
}

// downloadChunk writes part of the book starting at offset to file and returns number of bytes written
func downloadChunk(client *http.Client, file *os.File, downloadDetails model.BookDownloadDetails, offset uint64) (uint64, error) {
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return 0, err
	}
	request, err := http.NewRequest(http.MethodGet, downloadDetails.Url, nil)
	if err != nil {
		return 0, err
	}
	end := min(offset+downloadChunkSize, downloadDetails.Size) // exclusive
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end-1))
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusPartialContent:
		end, err = checkContentRange(response.Header.Get("Content-Range"), offset, end, downloadDetails.Size)
		if err != nil {
			return 0, err
		}
	case http.StatusOK:
		// server ignored requested range and sends whole book - correct only at the beginning
		if offset != 0 {
			return 0, fmt.Errorf("received whole book instead of part starting at byte %d", offset)
		}
		end = downloadDetails.Size
	default:
		return 0, fmt.Errorf("unexpected download response status: %s", response.Status)
	}
	expected := end - offset
	written, err := io.Copy(file, io.LimitReader(response.Body, int64(expected)))
	if err == nil && uint64(written) < expected {
		err = fmt.Errorf("received %d of %d bytes", written, expected)
	}
	return uint64(written), err
}

// checkContentRange verifies that server sends requested part of the book and returns its (exclusive) end
func checkContentRange(contentRange string, offset, requestedEnd, size uint64) (uint64, error) {
	var first, last, total uint64
	if _, err := fmt.Sscanf(contentRange, "bytes %d-%d/%d", &first, &last, &total); err != nil {
		return 0, fmt.Errorf("invalid Content-Range header: %q", contentRange)
	}
	if first != offset || last < first || last >= requestedEnd || total != size {
		return 0, fmt.Errorf("received bytes %d-%d/%d, requested %d-%d/%d", first, last, total, offset, requestedEnd-1, size)
	}
	return last + 1, nil
}
