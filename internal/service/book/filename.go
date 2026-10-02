package book

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tp86/legimi-go/internal/model"
)

// title of book hidden by Legimi and not remembered
const unknownTitle = "(title unknown)"

// maxTitleBytes keeps file name within 255 bytes limit of file systems (also with " (id).mobi.part")
const maxTitleBytes = 200

// characters not allowed in file names on Kindle (FAT32) and other systems
var fileNameReplacer = strings.NewReplacer(
	":", " -",
	"\"", "'",
	"/", "-",
	"\\", "-",
	"|", "-",
	"?", "",
	"*", "",
	"<", "",
	">", "",
)

// bookFileName returns "<title> (<id>).mobi", or "<id>.mobi" when title is unknown
func bookFileName(book model.BookMetadata) string {
	title := sanitizeTitle(book.Title)
	if title == "" || book.Title == unknownTitle {
		return fmt.Sprintf("%d.mobi", book.Id)
	}
	return fmt.Sprintf("%s (%d).mobi", title, book.Id)
}

func sanitizeTitle(title string) string {
	title = fileNameReplacer.Replace(title)
	// whitespace (also tabs and new lines) becomes single space, other control characters are removed
	title = strings.Join(strings.Fields(title), " ")
	title = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, title)
	if len(title) > maxTitleBytes {
		title = title[:maxTitleBytes]
		// don't cut multi-byte character in half
		for !utf8.ValidString(title) {
			title = title[:len(title)-1]
		}
	}
	// trailing dots and spaces are not allowed on FAT32
	return strings.TrimRight(title, ". ")
}
