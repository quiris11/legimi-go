package session

import (
	"github.com/tp86/legimi-go/internal/api"
	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/service"
)

type defaultSessionService struct {
	accountService service.Account
	client         api.Client
}

func (ss defaultSessionService) GetSession() (model.Session, error) {
	login, password := ss.accountService.GetCredentials()
	kindleId, err := ss.accountService.GetKindleId()
	if err != nil {
		return model.Session{}, err
	}
	var session model.Session
	err = ss.client.Exchange(model.NewGetSessionRequest(login, password, kindleId), &session)
	if err != nil {
		return model.Session{}, err
	}
	ss.accountService.SaveCredentials()
	return session, nil
}
