package presenter

import (
	"fmt"

	"github.com/tp86/legimi-go/internal/model"
)

type defaultBookListPresenter struct{}

func (defaultBookListPresenter) Present(bookList []model.BookMetadata, downloadLimit model.DownloadLimit) {
	for _, book := range bookList {
		fmt.Printf("%8d: \"%s\", %s, downloaded: %t\n", book.Id, book.Title, book.Author, book.Downloaded)
	}
	if downloadLimit.IsKnown() {
		fmt.Printf("\nDownloads left: %d of %d\n", downloadLimit.Left, downloadLimit.Max)
	} else {
		fmt.Println("\nDownloads left: unknown (is your Legimi package active?)")
	}
}

type defaultBookDownloadPresenter struct{}

func (defaultBookDownloadPresenter) Start(book model.BookMetadata) {
	fmt.Printf("Downloading book %d: \"%s\" ", book.Id, book.Title)
}

func (defaultBookDownloadPresenter) Part(book model.BookMetadata) {
	fmt.Print(".")
}

func (defaultBookDownloadPresenter) End(book model.BookMetadata) {
	fmt.Println(" done")
}

func (defaultBookDownloadPresenter) Wait(book model.BookMetadata) {
	fmt.Printf("Waiting for book %d: \"%s\" to be ready for download.\n", book.Id, book.Title)
}
