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
	if kindleId != 0 {
		return kindleId, nil
	}
	serialNumber, err := as.getKindleSerialNumber()
	if err != nil {
		return 0, err
	}
	return as.register(serialNumber)
}

// RefreshDevice registers Kindle again. Legimi hides book from shelf listing for device once its download
// is requested; registration resets device state, so such books are listed again.
func (as *defaultAccountService) RefreshDevice() (uint64, error) {
	previousKindleId := as.accountRepository.GetKindleId()
	serialNumber, err := as.getKindleSerialNumber()
	if err != nil {
		return 0, err
	}
	kindleId, err := as.register(serialNumber)
	if err == nil && previousKindleId != 0 && kindleId != previousKindleId {
		fmt.Fprintf(os.Stderr, "Warning: Legimi assigned new Kindle id %d (previously %d), check Kindle serial number in configuration file\n",
			kindleId, previousKindleId)
	}
	return kindleId, err
}

func (as *defaultAccountService) getKindleSerialNumber() (string, error) {
	if serialNumber := as.accountRepository.GetKindleSerialNumber(); serialNumber != "" {
		return serialNumber, nil
	}
	fmt.Print("Enter Kindle Serial Number: ")
	var serialNumber string
	if _, err := fmt.Scanln(&serialNumber); err != nil {
		return "", fmt.Errorf("couldn't read Kindle Serial Number: %v", err)
	}
	return serialNumber, nil
}

// register queries Legimi for Kindle id and stores it with serial number
func (as *defaultAccountService) register(serialNumber string) (uint64, error) {
	var registered model.Register
	login, password := as.GetCredentials()
	err := as.client.Exchange(model.NewRegisterRequest(login, password, serialNumber), &registered)
	if err != nil {
		return 0, fmt.Errorf("couldn't register Kindle: %v", err)
	}
	if registered.KindleId == 0 {
		return 0, fmt.Errorf("couldn't register Kindle: no Kindle id received")
	}
	// successful registration confirms credentials
	as.SaveCredentials()
	as.accountRepository.SaveKindleId(registered.KindleId)
	as.accountRepository.SaveKindleSerialNumber(serialNumber)
	return registered.KindleId, nil
}
