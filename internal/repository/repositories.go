package repository

type Account interface {
	GetLogin() string
	GetPassword() string
	GetKindleId() uint64
	SaveLogin(login string)
	SavePassword(password string) error
	SaveKindleId(kindleId uint64)
}
