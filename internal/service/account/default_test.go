package account

import (
	"testing"

	"github.com/tp86/legimi-go/internal/api/protocol"
	"github.com/tp86/legimi-go/internal/model"
	"github.com/tp86/legimi-go/internal/repository/account"
)

type registerClient struct {
	kindleId      uint64
	registrations int
}

func (c *registerClient) Exchange(request protocol.Request, response protocol.Response) error {
	if registered, ok := response.(*model.Register); ok {
		c.registrations++
		registered.KindleId = c.kindleId
	}
	return nil
}

type credentials struct{}

func (credentials) GetLogin() string    { return "user@example.com" }
func (credentials) GetPassword() string { return "secret" }

func TestRefreshUsesStoredSerialNumber(t *testing.T) {
	repository := &account.MemoryAccountRepository{KindleId: 1234567, KindleSerialNumber: "G000000000000000"}
	client := &registerClient{kindleId: 1234567}
	service := DefaultService(repository, client, credentials{})

	kindleId, err := service.RefreshDevice()
	if err != nil || kindleId != 1234567 || client.registrations != 1 {
		t.Errorf("kindle id %d, error %v, registrations %d", kindleId, err, client.registrations)
	}
}

func TestRegistrationStoresSerialNumberForRefresh(t *testing.T) {
	repository := &account.MemoryAccountRepository{KindleSerialNumber: "G000000000000000"}
	client := &registerClient{kindleId: 123}
	service := DefaultService(repository, client, credentials{})

	kindleId, err := service.GetKindleId()
	if err != nil || kindleId != 123 || repository.KindleId != 123 || repository.KindleSerialNumber != "G000000000000000" {
		t.Errorf("kindle id %d, error %v, repository %+v", kindleId, err, repository)
	}
	// stored id is used without registration
	service.GetKindleId()
	if client.registrations != 1 {
		t.Errorf("registrations %d", client.registrations)
	}
}

func TestRefreshFailsWithoutKindleId(t *testing.T) {
	repository := &account.MemoryAccountRepository{KindleSerialNumber: "G000000000000000"}
	service := DefaultService(repository, &registerClient{kindleId: 0}, credentials{})
	if _, err := service.RefreshDevice(); err == nil {
		t.Error("expected error")
	}
}
