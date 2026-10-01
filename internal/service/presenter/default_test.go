package presenter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/tp86/legimi-go/internal/model"
)

func TestBookListIsGroupedAndSortedByAuthorAndTitle(t *testing.T) {
	books := []model.BookMetadata{
		{Id: 1, Author: "Zofia Nałkowska", Title: "Granica", Downloaded: false},
		{Id: 2, Author: "Łukasz Orbitowski", Title: "Kult", Downloaded: true},
		{Id: 3, Author: "Bolesław Prus", Title: "Lalka", Downloaded: false},
		{Id: 4, Author: "Łukasz Orbitowski", Title: "Inna dusza", Downloaded: true},
		{Id: 5, Author: "lem Stanisław", Title: "Solaris", Downloaded: true},
		{Id: 6, Author: "Bolesław Prus", Title: "Emancypantki", Downloaded: false},
	}
	var out bytes.Buffer
	presentBookList(&out, books, model.DownloadLimit{Left: 3, Max: 10})

	expected := `Downloaded (3):
       5: lem Stanisław - "Solaris"
       4: Łukasz Orbitowski - "Inna dusza"
       2: Łukasz Orbitowski - "Kult"

Not downloaded (3):
       6: Bolesław Prus - "Emancypantki"
       3: Bolesław Prus - "Lalka"
       1: Zofia Nałkowska - "Granica"

Downloads left: 3 of 10
`
	if out.String() != expected {
		t.Errorf("got:\n%s\nexpected:\n%s", out.String(), expected)
	}
}

func TestBookListShowsLocalDownloads(t *testing.T) {
	downloaded := time.Date(2026, 1, 15, 18, 30, 0, 0, time.Local)
	books := []model.BookMetadata{
		{Id: 1, Author: "A", Title: "Listed", Downloaded: true, LastDownloaded: downloaded},
		{Id: 2, Author: "B", Title: "Hidden", Downloaded: true, LastDownloaded: downloaded, Hidden: true},
		{Id: 3, Author: "C", Title: "Hidden, not downloaded", Hidden: true},
	}
	var out bytes.Buffer
	presentBookList(&out, books, model.DownloadLimit{Left: 10, Max: 10})
	for _, expected := range []string{
		`1: A - "Listed" [downloaded with legimi-go 2026-01-15 18:30]` + "\n",
		`2: B - "Hidden" [downloaded with legimi-go 2026-01-15 18:30] [hidden by Legimi after download request]` + "\n",
		"Not downloaded (1):\n       3: C - \"Hidden, not downloaded\" [hidden by Legimi after download request]\n",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("output doesn't contain %q:\n%s", expected, out.String())
		}
	}
}

func TestEmptyBookList(t *testing.T) {
	var out bytes.Buffer
	presentBookList(&out, nil, model.DownloadLimit{Left: 4294967295, Max: 4294967295})
	expected := "No books on shelf.\nDownloads left: unknown (is your Legimi package active?)\n"
	if out.String() != expected {
		t.Errorf("got:\n%q\nexpected:\n%q", out.String(), expected)
	}
}
