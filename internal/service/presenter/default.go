package presenter

import (
	"fmt"
	"io"
	"os"
	"slices"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/tp86/legimi-go/internal/model"
)

type defaultBookListPresenter struct{}

func (defaultBookListPresenter) Present(bookList []model.BookMetadata, downloadLimit model.DownloadLimit) {
	presentBookList(os.Stdout, bookList, downloadLimit)
}

func presentBookList(w io.Writer, bookList []model.BookMetadata, downloadLimit model.DownloadLimit) {
	if len(bookList) == 0 {
		fmt.Fprintln(w, "No books on shelf.")
	}
	var downloaded, notDownloaded []model.BookMetadata
	for _, book := range bookList {
		if book.Downloaded {
			downloaded = append(downloaded, book)
		} else {
			notDownloaded = append(notDownloaded, book)
		}
	}
	presentSection(w, "Downloaded", downloaded)
	presentSection(w, "Not downloaded", notDownloaded)
	hidden := 0
	for _, book := range bookList {
		if book.Hidden {
			hidden++
		}
	}
	if hidden > 0 {
		fmt.Fprintf(w, "%d book(s) hidden by Legimi after download request, use refresh command to list them again.\n", hidden)
	}
	if downloadLimit.IsKnown() {
		fmt.Fprintf(w, "Downloads left: %d of %d\n", downloadLimit.Left, downloadLimit.Max)
	} else {
		fmt.Fprintln(w, "Downloads left: unknown (is your Legimi package active?)")
	}
}

func presentSection(w io.Writer, header string, books []model.BookMetadata) {
	if len(books) == 0 {
		return
	}
	sortByAuthorAndTitle(books)
	fmt.Fprintf(w, "%s (%d):\n", header, len(books))
	for _, book := range books {
		fmt.Fprintf(w, "%8d: %s - \"%s\"%s\n", book.Id, book.Author, book.Title, bookNotes(book))
	}
	fmt.Fprintln(w)
}

// bookNotes describes downloads made with this program
func bookNotes(book model.BookMetadata) string {
	var notes string
	if !book.LastDownloaded.IsZero() {
		notes += " [downloaded with legimi-go " + book.LastDownloaded.Local().Format("2006-01-02 15:04") + "]"
	}
	if book.Hidden {
		notes += " [hidden by Legimi after download request]"
	}
	return notes
}

func sortByAuthorAndTitle(books []model.BookMetadata) {
	// Polish alphabetical order, e.g. "Ł" between "L" and "M"
	collator := collate.New(language.Polish, collate.IgnoreCase)
	slices.SortStableFunc(books, func(a, b model.BookMetadata) int {
		if c := collator.CompareString(a.Author, b.Author); c != 0 {
			return c
		}
		return collator.CompareString(a.Title, b.Title)
	})
}

type defaultBookDownloadPresenter struct{}

func (defaultBookDownloadPresenter) Start(book model.BookMetadata) {
	fmt.Printf("Downloading book %d: \"%s\" ", book.Id, book.Title)
}

func (defaultBookDownloadPresenter) Part(book model.BookMetadata) {
	fmt.Print(".")
}

func (defaultBookDownloadPresenter) End(book model.BookMetadata, fileName string) {
	fmt.Printf(" done: %s\n", fileName)
}

func (defaultBookDownloadPresenter) Fail(book model.BookMetadata) {
	fmt.Println(" failed")
}

func (defaultBookDownloadPresenter) Skip(book model.BookMetadata, reason error) {
	if book.Title != "" {
		fmt.Printf("Book %d \"%s\" not downloaded: %v\n", book.Id, book.Title, reason)
	} else {
		fmt.Printf("Book %d not downloaded: %v\n", book.Id, reason)
	}
}

func (defaultBookDownloadPresenter) Wait(book model.BookMetadata) {
	fmt.Printf("Waiting for book %d: \"%s\" to be ready for download.\n", book.Id, book.Title)
}
