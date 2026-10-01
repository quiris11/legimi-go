package presenter

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/tp86/legimi-go/internal/model"
)

const (
	alternateScreenOn  = "\x1b[?1049h"
	alternateScreenOff = "\x1b[?1049l"
	cursorHide         = "\x1b[?25l"
	cursorShow         = "\x1b[?25h"
	cursorHomeClear    = "\x1b[H\x1b[2J"
)

type terminalBookSelector struct{}

// Select lets user choose books interactively and returns their ids (none if user quits)
func (terminalBookSelector) Select(books []model.BookMetadata, downloadLimit model.DownloadLimit) ([]uint64, error) {
	if len(books) == 0 {
		fmt.Println("No books on shelf.")
		return nil, nil
	}
	in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	if !term.IsTerminal(in) || !term.IsTerminal(out) {
		return nil, fmt.Errorf("select command requires interactive terminal, use download command instead")
	}
	state, err := term.MakeRaw(in)
	if err != nil {
		return nil, err
	}
	fmt.Print(alternateScreenOn + cursorHide)
	defer func() {
		fmt.Print(cursorShow + alternateScreenOff)
		term.Restore(in, state)
	}()

	s := newSelector(books, downloadLimit)
	buffer := make([]byte, 256)
	for s.result == selecting {
		width, height, err := term.GetSize(out)
		if err != nil {
			width, height = 80, 24
		}
		// in raw mode new line doesn't return carriage
		fmt.Print(cursorHomeClear + strings.Join(s.render(width, height), "\r\n"))
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return nil, err
		}
		for _, event := range parseKeys(buffer[:n]) {
			s.handle(event, listHeight(height))
		}
	}
	if s.result == cancelled {
		return nil, nil
	}
	return s.selectedIds(), nil
}
