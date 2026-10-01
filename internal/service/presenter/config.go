package presenter

import "github.com/tp86/legimi-go/internal/service"

func DefaultBookListPresenter() service.BookListPresenter {
	return defaultBookListPresenter{}
}

func DefaultBookSelector() service.BookSelector {
	return terminalBookSelector{}
}

func DefaultBookDownloadPresenter() service.DownloadPresenter {
	return defaultBookDownloadPresenter{}
}
