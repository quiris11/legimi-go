package main

import (
	"github.com/tp86/legimi-go/internal/api"
	"github.com/tp86/legimi-go/internal/commands"
	ar "github.com/tp86/legimi-go/internal/repository/account"
	br "github.com/tp86/legimi-go/internal/repository/book"
	as "github.com/tp86/legimi-go/internal/service/account"
	"github.com/tp86/legimi-go/internal/service/book"
	"github.com/tp86/legimi-go/internal/service/presenter"
	"github.com/tp86/legimi-go/internal/service/session"
)

func configure() {
	accountRepository := ar.GetFileRepository(commands.Options)
	apiClient := api.GetClient(commands.Options)
	accountService := as.DefaultService(accountRepository, apiClient, commands.Options)
	sessionService := session.DefaultService(accountService, apiClient)
	bookDownloadPresenter := presenter.DefaultBookDownloadPresenter()
	// command line option takes precedence over configuration file
	downloadDirectory := commands.Options.GetDownloadDirectory()
	if downloadDirectory == "" {
		downloadDirectory = accountRepository.GetDownloadDirectory()
	}
	bookService := book.DefaultService(sessionService, apiClient, bookDownloadPresenter, br.GetFileRepository(commands.Options), downloadDirectory)
	commands.BookLister = bookService
	commands.BookDownloader = bookService
	commands.BookListPresenter = presenter.DefaultBookListPresenter()
	commands.BookSelector = presenter.DefaultBookSelector()
	commands.DeviceRefresher = accountService
}
