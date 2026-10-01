package presenter

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/tp86/legimi-go/internal/model"
)

type keyType int

const (
	keyRune keyType = iota
	keyUp
	keyDown
	keyPageUp
	keyPageDown
	keyHome
	keyEnd
	keyToggle
	keyEnter
	keyBackspace
	keyClearFilter
	keyQuit
)

type keyEvent struct {
	key  keyType
	char rune
}

// parseKeys translates bytes read from terminal in raw mode into key events
func parseKeys(input []byte) []keyEvent {
	sequences := map[string]keyType{
		"\x1b[A": keyUp, "\x1bOA": keyUp,
		"\x1b[B": keyDown, "\x1bOB": keyDown,
		"\x1b[5~": keyPageUp, "\x1b[6~": keyPageDown,
		"\x1b[H": keyHome, "\x1bOH": keyHome, "\x1b[1~": keyHome,
		"\x1b[F": keyEnd, "\x1bOF": keyEnd, "\x1b[4~": keyEnd,
	}
	var events []keyEvent
	for len(input) > 0 {
		if input[0] == 0x1b {
			if len(input) == 1 {
				events = append(events, keyEvent{key: keyQuit})
				break
			}
			found := false
			for sequence, key := range sequences {
				if strings.HasPrefix(string(input), sequence) {
					events = append(events, keyEvent{key: key})
					input = input[len(sequence):]
					found = true
					break
				}
			}
			if !found {
				// unsupported escape sequence - ignore rest of input
				break
			}
			continue
		}
		char, size := utf8.DecodeRune(input)
		input = input[size:]
		switch char {
		case 0x03: // Ctrl+C
			events = append(events, keyEvent{key: keyQuit})
		case '\r', '\n':
			events = append(events, keyEvent{key: keyEnter})
		case 0x7f, 0x08:
			events = append(events, keyEvent{key: keyBackspace})
		case 0x15: // Ctrl+U
			events = append(events, keyEvent{key: keyClearFilter})
		case ' ', '\t':
			events = append(events, keyEvent{key: keyToggle})
		default:
			if unicode.IsPrint(char) {
				events = append(events, keyEvent{key: keyRune, char: char})
			}
		}
	}
	return events
}

type selectorResult int

const (
	selecting selectorResult = iota
	confirmed
	cancelled
)

// selector holds state of interactive book selection, independent of terminal
type selector struct {
	books         []model.BookMetadata
	searchText    []string // normalized author and title of each book
	downloadLimit model.DownloadLimit
	filter        []rune
	visible       []int // indexes of books matching filter
	cursor        int   // index in visible
	top           int   // index in visible of first displayed book
	selected      map[uint64]bool
	confirming    bool
	// book selected implicitly by Enter, unselected when download is not confirmed
	autoSelected *uint64
	result       selectorResult
}

func newSelector(books []model.BookMetadata, downloadLimit model.DownloadLimit) *selector {
	// not downloaded books first, each group sorted like in book list
	var notDownloaded, downloaded []model.BookMetadata
	for _, book := range books {
		if book.Downloaded {
			downloaded = append(downloaded, book)
		} else {
			notDownloaded = append(notDownloaded, book)
		}
	}
	sortByAuthorAndTitle(notDownloaded)
	sortByAuthorAndTitle(downloaded)
	s := &selector{
		books:         append(notDownloaded, downloaded...),
		downloadLimit: downloadLimit,
		selected:      make(map[uint64]bool),
	}
	for _, book := range s.books {
		s.searchText = append(s.searchText, normalize(book.Author+" "+book.Title))
	}
	s.applyFilter()
	return s
}

var removeDiacritics = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// normalize makes text lower case and without diacritics, so "zolc" matches "Żółć"
func normalize(text string) string {
	text = strings.NewReplacer("ł", "l", "Ł", "l").Replace(text)
	text, _, _ = transform.String(removeDiacritics, text)
	return strings.ToLower(text)
}

func (s *selector) applyFilter() {
	filter := normalize(string(s.filter))
	s.visible = s.visible[:0]
	for i, text := range s.searchText {
		if strings.Contains(text, filter) {
			s.visible = append(s.visible, i)
		}
	}
	s.cursor, s.top = 0, 0
}

func (s *selector) moveCursor(by int) {
	s.cursor = max(0, min(len(s.visible)-1, s.cursor+by))
}

func (s *selector) currentBook() (model.BookMetadata, bool) {
	if len(s.visible) == 0 {
		return model.BookMetadata{}, false
	}
	return s.books[s.visible[s.cursor]], true
}

// selectedIds returns selected books' ids in display order
func (s *selector) selectedIds() []uint64 {
	var ids []uint64
	for _, book := range s.books {
		if s.selected[book.Id] {
			ids = append(ids, book.Id)
		}
	}
	return ids
}

func (s *selector) selectedNotDownloaded() int {
	count := 0
	for _, book := range s.books {
		if s.selected[book.Id] && !book.Downloaded {
			count++
		}
	}
	return count
}

func (s *selector) handle(event keyEvent, pageSize int) {
	if s.confirming {
		// explicit answer required, so that accidental double Enter doesn't use downloads
		if event.key == keyRune && strings.ContainsRune("yYtT", event.char) {
			s.result = confirmed
			return
		}
		s.confirming = false
		if s.autoSelected != nil {
			delete(s.selected, *s.autoSelected)
			s.autoSelected = nil
		}
		return
	}
	switch event.key {
	case keyUp:
		s.moveCursor(-1)
	case keyDown:
		s.moveCursor(1)
	case keyPageUp:
		s.moveCursor(-pageSize)
	case keyPageDown:
		s.moveCursor(pageSize)
	case keyHome:
		s.moveCursor(-len(s.visible))
	case keyEnd:
		s.moveCursor(len(s.visible))
	case keyToggle:
		if book, ok := s.currentBook(); ok {
			if s.selected[book.Id] {
				delete(s.selected, book.Id)
			} else {
				s.selected[book.Id] = true
			}
			s.moveCursor(1)
		}
	case keyEnter:
		// without explicit selection, book under cursor is downloaded
		if len(s.selected) == 0 {
			if book, ok := s.currentBook(); ok {
				s.selected[book.Id] = true
				s.autoSelected = &book.Id
			}
		}
		if len(s.selected) > 0 {
			s.confirming = true
		}
	case keyRune:
		s.filter = append(s.filter, event.char)
		s.applyFilter()
	case keyBackspace:
		if len(s.filter) > 0 {
			s.filter = s.filter[:len(s.filter)-1]
			s.applyFilter()
		}
	case keyClearFilter:
		s.filter = nil
		s.applyFilter()
	case keyQuit:
		s.result = cancelled
	}
}

const (
	styleReset    = "\x1b[0m"
	styleReverse  = "\x1b[7m"
	styleDim      = "\x1b[2m"
	styleBold     = "\x1b[1m"
	headerLines   = 2
	footerLines   = 3
	minListHeight = 1
)

func listHeight(height int) int {
	return max(minListHeight, height-headerLines-footerLines)
}

// render returns screen lines for terminal of given size
func (s *selector) render(width, height int) []string {
	listHeight := listHeight(height)
	// keep cursor on screen
	if s.cursor < s.top {
		s.top = s.cursor
	}
	if s.cursor >= s.top+listHeight {
		s.top = s.cursor - listHeight + 1
	}

	lines := []string{
		fit(fmt.Sprintf("Filter: %s_   (%d of %d books)", string(s.filter), len(s.visible), len(s.books)), width),
		"",
	}
	for row := 0; row < listHeight; row++ {
		i := s.top + row
		if i >= len(s.visible) {
			lines = append(lines, "")
			continue
		}
		book := s.books[s.visible[i]]
		mark := "[ ]"
		if s.selected[book.Id] {
			mark = "[x]"
		}
		text := fmt.Sprintf("  %s %s - \"%s\"", mark, book.Author, book.Title)
		if book.Downloaded && book.LastDownloaded.IsZero() {
			text += " (downloaded)"
		}
		text += bookNotes(book)
		text = fit(text, width)
		switch {
		case i == s.cursor:
			text = styleReverse + text + styleReset
		case book.Downloaded:
			text = styleDim + text + styleReset
		}
		lines = append(lines, text)
	}

	status := fmt.Sprintf("Selected: %d", len(s.selected))
	if notDownloaded := s.selectedNotDownloaded(); notDownloaded > 0 {
		status += fmt.Sprintf(" (%d not downloaded yet)", notDownloaded)
	}
	if s.downloadLimit.IsKnown() {
		status += fmt.Sprintf("   Downloads left: %d of %d", s.downloadLimit.Left, s.downloadLimit.Max)
		if uint32(s.selectedNotDownloaded()) > s.downloadLimit.Left {
			status += "   NOT ENOUGH DOWNLOADS LEFT"
		}
	}
	help := "Up/Down move  Space select  type to filter  Enter download  Esc quit"
	if s.confirming {
		help = styleBold + fmt.Sprintf("Download %d book(s)? [y/n]", len(s.selected)) + styleReset
	}
	return append(lines, "", fit(status, width), fit(help, width))
}

// fit truncates text to given number of characters
func fit(text string, width int) string {
	if utf8.RuneCountInString(text) <= width {
		return text
	}
	if width <= 1 {
		return string([]rune(text)[:max(0, width)])
	}
	return string([]rune(text)[:width-1]) + "…"
}
