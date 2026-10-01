package options

type Credentials interface {
	GetLogin() string
	GetPassword() string
}

type Configuration interface {
	GetFile() string
}

type Download interface {
	GetDownloadDirectory() string
}

type Debugging interface {
	IsDebug() bool
}
