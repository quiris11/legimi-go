package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/tp86/legimi-go/internal/paths"
	"github.com/tp86/legimi-go/internal/service"
	"github.com/tp86/legimi-go/internal/usecase"
)

type Command struct {
	name        string
	args        string
	description string
	Run         func() error
}

var noneCommand = Command{
	Run: func() error {
		return fmt.Errorf("this should never be called")
	},
}

var (
	Commands = []Command{
		{name: "list", Run: listBooks, description: "list books on shelf"},
		{name: "download", args: "id ...", Run: downloadBooks, description: "download book(s) with given id(s)"},
		{name: "select", Run: selectBooks, description: "select book(s) to download from interactive list"},
		{name: "dir", args: "[directory]", Run: downloadDirectory, description: "show or set directory for downloaded books (\".\" for current directory)"},
		{name: "refresh", Run: refreshDevice, description: "register Kindle again, so that books hidden by Legimi after download are listed again"},
		{name: "version", Run: printVersion, description: "print version of script"},
	}
)

func ParseCommandLine() (Command, error) {
	configureFlags()
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		names := make([]string, len(Commands))
		for i, command := range Commands {
			names[i] = command.name
		}
		return noneCommand, fmt.Errorf("expected one of commands: %s\n", strings.Join(names, ", "))
	}
	commandName := args[0]
	if command, ok := findCommand(commandName); ok {
		return command, nil
	}
	return noneCommand, fmt.Errorf("unsupported command: %s", commandName)
}

func findCommand(name string) (Command, bool) {
	for _, command := range Commands {
		if command.name == name {
			return command, true
		}
	}
	return noneCommand, false
}

var (
	BookLister        usecase.BookLister
	BookListPresenter service.BookListPresenter
	BookDownloader    usecase.BookDownloader
	BookSelector      service.BookSelector
	DeviceRefresher   usecase.DeviceRefresher
	DownloadDirectory DownloadDirectorySetting
)

type DownloadDirectorySetting interface {
	GetDownloadDirectory() string
	SaveDownloadDirectory(directory string)
}

func listBooks() error {
	bookList, downloadLimit, err := BookLister.ListBooks()
	if err != nil {
		return err
	}
	BookListPresenter.Present(bookList, downloadLimit)
	return nil
}

func selectBooks() error {
	// fail before user selects books
	if err := BookDownloader.CheckDownloadDirectory(); err != nil {
		return err
	}
	bookList, downloadLimit, err := BookLister.ListBooks()
	if err != nil {
		return err
	}
	bookIds, err := BookSelector.Select(bookList, downloadLimit)
	if err != nil || len(bookIds) == 0 {
		return err
	}
	return BookDownloader.DownloadBooks(bookIds)
}

func downloadBooks() error {
	ids := flag.Args()[1:]
	if len(ids) == 0 {
		return fmt.Errorf("no book id provided")
	}
	bookIds := make([]uint64, len(ids))
	for i, id := range ids {
		v, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			err := err.(*strconv.NumError)
			return fmt.Errorf("couldn't parse id '%s': %s", id, err.Err)
		}
		bookIds[i] = v
	}
	return BookDownloader.DownloadBooks(bookIds)
}

func downloadDirectory() error {
	args := flag.Args()[1:]
	switch {
	case len(args) == 0:
		if directory := DownloadDirectory.GetDownloadDirectory(); directory != "" {
			fmt.Printf("Books are downloaded to: %s\n", directory)
		} else {
			fmt.Println("Books are downloaded to current directory.")
		}
		if Options.GetDownloadDirectory() != "" {
			fmt.Printf("For this run --dir option sets: %s\n", Options.GetDownloadDirectory())
		}
		return nil
	case len(args) > 1:
		return fmt.Errorf("expected one directory, use quotes for path with spaces")
	case args[0] == ".":
		DownloadDirectory.SaveDownloadDirectory("")
		fmt.Println("Books will be downloaded to current directory.")
		return nil
	}
	directory, err := filepath.Abs(paths.ExpandHome(args[0]))
	if err != nil {
		return err
	}
	DownloadDirectory.SaveDownloadDirectory(directory)
	fmt.Printf("Books will be downloaded to: %s\n", directory)
	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		fmt.Println("Warning: this directory doesn't exist now, downloads will fail until it does (e.g. until Kindle is connected).")
	}
	return nil
}

func refreshDevice() error {
	kindleId, err := DeviceRefresher.RefreshDevice()
	if err != nil {
		return err
	}
	fmt.Printf("Kindle (id %d) registered again, books hidden by Legimi after download should be listed again.\n", kindleId)
	return nil
}

func printVersion() error {
	if info, ok := debug.ReadBuildInfo(); !ok {
		return fmt.Errorf("Error getting version info")
	} else {
		fmt.Printf("Legimi-go version: %s\n", info.Main.Version)
	}
	return nil
}
