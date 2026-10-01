package book

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/tp86/legimi-go/internal/api/protocol"
)

// versionsClient answers download details requests: preparing for versions in preparing,
// details for other versions up to current and "version not available" above it
type versionsClient struct {
	current   uint64
	preparing map[uint64]bool
	requested []uint64
}

func (c *versionsClient) Exchange(request protocol.Request, response protocol.Response) error {
	var buf bytes.Buffer
	request.Encode(&buf)
	// request starts with book id and version
	version := binary.LittleEndian.Uint64(buf.Bytes()[8:16])
	c.requested = append(c.requested, version)
	switch {
	case version > c.current:
		return protocol.ErrorResponse{Type: protocol.BookVersionNotAvailableError}
	case c.preparing[version]:
		return protocol.ErrorResponse{Type: protocol.BookDownloadDetailsPreparingError}
	}
	return nil
}

func TestCurrentVersionIsFound(t *testing.T) {
	tests := []versionsClient{
		{current: 1},
		{current: 2, preparing: map[uint64]bool{1: true}},
		{current: 3, preparing: map[uint64]bool{3: true}},
	}
	for _, client := range tests {
		bs := defaultBookService{client: &client}
		version, err := bs.findCurrentVersion("session", 1000001)
		if err != nil || version != client.current {
			t.Errorf("current %d: found %d, error %v, requested %v", client.current, version, err, client.requested)
		}
	}
}

func TestBookWithoutVersionsIsNotAvailable(t *testing.T) {
	bs := defaultBookService{client: &versionsClient{current: 0}}
	if _, err := bs.findCurrentVersion("session", 1); err == nil {
		t.Error("expected error")
	}
}
