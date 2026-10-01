package account

type MemoryAccountRepository struct {
	Login, Password    string
	KindleId           uint64
	KindleSerialNumber string
	DownloadDirectory  string
}

func (mar MemoryAccountRepository) GetDownloadDirectory() string {
	return mar.DownloadDirectory
}

func (mar MemoryAccountRepository) GetKindleSerialNumber() string {
	return mar.KindleSerialNumber
}

func (mar *MemoryAccountRepository) SaveKindleSerialNumber(serialNumber string) {
	mar.KindleSerialNumber = serialNumber
}

func (mar MemoryAccountRepository) GetLogin() string {
	return mar.Login
}

func (mar MemoryAccountRepository) GetPassword() string {
	return mar.Password
}

func (mar MemoryAccountRepository) GetKindleId() uint64 {
	return mar.KindleId
}

func (mar *MemoryAccountRepository) SaveLogin(login string) {
	mar.Login = login
}

func (mar *MemoryAccountRepository) SavePassword(password string) error {
	mar.Password = password
	return nil
}

func (mar *MemoryAccountRepository) SaveKindleId(id uint64) {
	mar.KindleId = id
}
