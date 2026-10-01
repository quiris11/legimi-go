package account

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/tp86/legimi-go/internal/api"
	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/options"
	"github.com/tp86/legimi-go/internal/repository"
)

// TODO handle errors

type defaultAccountService struct {
	accountRepository repository.Account
	client            api.Client
	userCredentials   options.Credentials
	login             string
	password          string
	// credentials entered by user are saved only after successful login
	loginEntered    bool
	passwordEntered bool
}

func (as *defaultAccountService) getLogin() string {
	if as.login != "" {
		return as.login
	}
	login := as.userCredentials.GetLogin()
	if login == "" {
		login = as.accountRepository.GetLogin()
	}
	if login == "" {
		fmt.Print("Enter legimi login: ")
		if _, err := fmt.Scanln(&login); err == nil {
			as.loginEntered = true
		}
	}
	as.login = login
	return login
}

func (as *defaultAccountService) getPassword() string {
	if as.password != "" {
		return as.password
	}
	password := as.userCredentials.GetPassword()
	if password == "" {
		password = as.accountRepository.GetPassword()
	}
	if password == "" {
		fmt.Print("Enter legimi password: ")
		if passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd())); err == nil {
			password = string(passwordBytes)
			as.passwordEntered = true
		}
		fmt.Println()
	}
	as.password = password
	return password
}

func (as *defaultAccountService) GetCredentials() (string, string) {
	login := as.getLogin()
	password := as.getPassword()
	return login, password
}

func (as *defaultAccountService) SaveCredentials() {
	if as.loginEntered {
		as.accountRepository.SaveLogin(as.login)
		as.loginEntered = false
	}
	if as.passwordEntered {
		if err := as.accountRepository.SavePassword(as.password); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: password not saved: %v\n", err)
		}
		as.passwordEntered = false
	}
}

func (as *defaultAccountService) GetKindleId() (uint64, error) {
	kindleId := as.accountRepository.GetKindleId()
	// if kindle id is not in repository
	if kindleId == 0 {
		// ask user for Kindle Serial No
		fmt.Print("Enter Kindle Serial Number: ")
		var kindleSerialNumber string
		if _, err := fmt.Scanln(&kindleSerialNumber); err != nil {
			return 0, fmt.Errorf("couldn't read Kindle Serial Number: %v", err)
		}
		// then query api for kindle id
		var registered model.Register
		login, password := as.GetCredentials()
		err := as.client.Exchange(model.NewRegisterRequest(login, password, kindleSerialNumber), &registered)
		if err != nil {
			return 0, fmt.Errorf("couldn't register Kindle: %v", err)
		}
		if registered.KindleId == 0 {
			return 0, fmt.Errorf("couldn't register Kindle: no Kindle id received")
		}
		// successful registration confirms credentials
		as.SaveCredentials()
		kindleId = registered.KindleId
		// and store result in repository
		as.accountRepository.SaveKindleId(kindleId)
	}
	return kindleId, nil
}
