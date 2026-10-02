package protocol

import "testing"

func TestErrorResponseMessages(t *testing.T) {
	tests := map[uint16]string{
		139:  "book is not available for download to Kindle (even with active package)",
		5143: "Legimi refused download (probably no downloads left in subscription period)",
		9999: "error response received: 9999",
	}
	for code, expected := range tests {
		if got := (ErrorResponse{Type: code}).Error(); got != expected {
			t.Errorf("%d: got %q, expected %q", code, got, expected)
		}
	}
}
