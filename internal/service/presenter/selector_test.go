package presenter

import (
	"slices"
	"strings"
	"testing"

	"github.com/tp86/legimi-go/internal/model"
)

var shelf = []model.BookMetadata{
	{Id: 1, Author: "Adam Mickiewicz", Title: "Pan Tadeusz", Downloaded: true},
	{Id: 2, Author: "Adam Mickiewicz", Title: "Konrad Wallenrod"},
	{Id: 3, Author: "Bolesław Prus", Title: "Lalka"},
	{Id: 4, Author: "Łukasz Górnicki", Title: "Dworzanin polski"},
	{Id: 5, Author: "Zofia Nałkowska", Title: "Granica", Downloaded: true},
}

func press(s *selector, input string) {
	for _, event := range parseKeys([]byte(input)) {
		s.handle(event, 10)
	}
}

func displayedIds(s *selector) []uint64 {
	var ids []uint64
	for _, i := range s.visible {
		ids = append(ids, s.books[i].Id)
	}
	return ids
}

func TestNotDownloadedBooksAreShownFirst(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{Left: 10, Max: 10})
	if ids := displayedIds(s); !slices.Equal(ids, []uint64{2, 3, 4, 1, 5}) {
		t.Errorf("displayed %v", ids)
	}
}

func TestFilterIgnoresCaseAndDiacritics(t *testing.T) {
	tests := map[string][]uint64{
		"mickiew":  {2, 1},
		"MICKIEW":  {2, 1},
		"lukasz":   {4},
		"nalkowsk": {5},
		"polski":   {4},
		"xyz":      nil,
	}
	for filter, expected := range tests {
		s := newSelector(shelf, model.DownloadLimit{})
		press(s, filter)
		if ids := displayedIds(s); !slices.Equal(ids, expected) {
			t.Errorf("filter %q: displayed %v, expected %v", filter, ids, expected)
		}
	}
}

func TestBackspaceAndClearFilter(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	press(s, "prusx")
	press(s, "\x7f")
	if ids := displayedIds(s); !slices.Equal(ids, []uint64{3}) {
		t.Errorf("after backspace displayed %v", ids)
	}
	press(s, "\x15")
	if len(s.visible) != len(shelf) {
		t.Errorf("after clearing filter displayed %v", displayedIds(s))
	}
}

func TestSelectWithSpaceAndConfirm(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	// select first, skip second, select third
	press(s, " \x1b[B ")
	press(s, "\r")
	if !s.confirming || s.result != selecting {
		t.Fatal("download should wait for confirmation")
	}
	press(s, "y")
	if s.result != confirmed || !slices.Equal(s.selectedIds(), []uint64{2, 4}) {
		t.Errorf("result %v, selected %v", s.result, s.selectedIds())
	}
}

func TestSelectionIsKeptWhileFiltering(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	press(s, "prus ")
	press(s, "\x15mickiew ")
	press(s, "\rt")
	if s.result != confirmed || !slices.Equal(s.selectedIds(), []uint64{2, 3}) {
		t.Errorf("result %v, selected %v", s.result, s.selectedIds())
	}
}

func TestEnterWithoutSelectionDownloadsCurrentBook(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	press(s, "\x1b[B\x1b[B\ry")
	if s.result != confirmed || !slices.Equal(s.selectedIds(), []uint64{4}) {
		t.Errorf("result %v, selected %v", s.result, s.selectedIds())
	}
}

func TestDownloadNotConfirmed(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	// double Enter must not confirm download; implicitly selected book is unselected
	press(s, "\r\r")
	if s.result != selecting || s.confirming || len(s.selected) != 0 {
		t.Errorf("result %v, confirming %v, selected %v", s.result, s.confirming, s.selectedIds())
	}
}

func TestQuit(t *testing.T) {
	for _, key := range []string{"\x1b", "\x03"} {
		s := newSelector(shelf, model.DownloadLimit{})
		press(s, " ")
		press(s, key)
		if s.result != cancelled {
			t.Errorf("key %q: result %v", key, s.result)
		}
	}
}

func TestCursorStaysWithinList(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	press(s, "\x1b[A\x1b[A")
	if s.cursor != 0 {
		t.Errorf("cursor %d after moving up at top", s.cursor)
	}
	press(s, "\x1b[6~\x1b[6~")
	if s.cursor != len(shelf)-1 {
		t.Errorf("cursor %d after page down", s.cursor)
	}
	press(s, "\x1b[H")
	if s.cursor != 0 {
		t.Errorf("cursor %d after home", s.cursor)
	}
}

func TestRenderShowsSelectionAndLimit(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{Left: 1, Max: 10})
	press(s, "  ")
	screen := strings.Join(s.render(200, 12), "\n")
	for _, expected := range []string{
		"Filter: _   (5 of 5 books)",
		`[x] Adam Mickiewicz - "Konrad Wallenrod"`,
		`[x] Bolesław Prus - "Lalka"`,
		`[ ] Adam Mickiewicz - "Pan Tadeusz" (downloaded)`,
		"Selected: 2 (2 not downloaded yet)   Downloads left: 1 of 10   NOT ENOUGH DOWNLOADS LEFT",
	} {
		if !strings.Contains(screen, expected) {
			t.Errorf("screen doesn't contain %q:\n%s", expected, screen)
		}
	}
}

func TestRenderScrollsToCursor(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	press(s, "\x1b[F")
	// height 7 leaves 2 lines for the list
	screen := strings.Join(s.render(200, 7), "\n")
	if !strings.Contains(screen, "Granica") || strings.Contains(screen, "Lalka") {
		t.Errorf("last book should be displayed:\n%s", screen)
	}
}

func TestLongLinesAreTruncated(t *testing.T) {
	s := newSelector(shelf, model.DownloadLimit{})
	for _, line := range s.render(20, 10) {
		line = strings.NewReplacer(styleReverse, "", styleDim, "", styleReset, "", styleBold, "").Replace(line)
		if n := len([]rune(line)); n > 20 {
			t.Errorf("line has %d characters: %q", n, line)
		}
	}
}
