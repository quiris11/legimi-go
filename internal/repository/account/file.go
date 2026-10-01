package account

import (
	"fmt"
	"os"
	"path"

	"github.com/tp86/legimi-go/internal/repository"
	"gopkg.in/ini.v1"
)

type fileAccountRepository struct {
	filePath string
	file     *ini.File
	config   *ini.Section
}

func newFileAccountRepository(configFile string) repository.Account {
	file, err := ini.Load(configFile)
	if err != nil {
		file = ini.Empty()
		os.MkdirAll(path.Dir(configFile), 0700)
	} else {
		// restrict permissions of configuration files created by previous versions
		os.Chmod(configFile, 0600)
	}
	far := &fileAccountRepository{
		filePath: configFile,
		file:     file,
		config:   file.Section(""),
	}
	if err != nil {
		far.save()
	}
	return far
}

// save writes configuration file readable only by its owner, as it contains credentials
func (far fileAccountRepository) save() error {
	f, err := os.OpenFile(far.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	_, err = far.file.WriteTo(f)
	return err
}

func (far fileAccountRepository) GetLogin() string {
	key, err := far.config.GetKey("login")
	if err != nil {
		return ""
	}
	return key.MustString("")
}

func (far fileAccountRepository) GetPassword() string {
	key, err := far.config.GetKey("password")
	if err != nil {
		return ""
	}
	return key.MustString("")
}

func (far fileAccountRepository) GetKindleId() uint64 {
	key, err := far.config.GetKey("kindleId")
	if err != nil {
		return 0
	}
	return key.MustUint64(0)
}

func (far fileAccountRepository) SaveLogin(login string) {
	key := far.config.Key("login")
	key.SetValue(login)
	far.save()
}

func (far fileAccountRepository) SavePassword(password string) {
	key := far.config.Key("password")
	key.SetValue(password)
	far.save()
}

func (far fileAccountRepository) SaveKindleId(kindleId uint64) {
	key := far.config.Key("kindleId")
	key.SetValue(fmt.Sprint(kindleId))
	far.save()
}
