package repository

import "github.com/tp86/legimi-go/internal/model"

type Book interface {
	Get(id uint64) (model.BookMetadata, bool)
	Save(books []model.BookMetadata) error
}

type Account interface {
	GetLogin() string
	GetPassword() string
	GetKindleId() uint64
	SaveLogin(login string)
	SavePassword(password string) error
	SaveKindleId(kindleId uint64)
}
