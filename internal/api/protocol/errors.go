package protocol

import "fmt"

const (
	BookDownloadDetailsPreparingError = 295
	// returned for book version higher than current one
	BookVersionNotAvailableError = 135
	// returned when no downloads are left in subscription period (observed, not documented)
	DownloadRefusedError = 5143
)

// TODO list known errors and their descriptions
var errorResponses = map[uint16]string{
	133:  "invalid credentials",
	163:  "invalid kindle id",
	135:  "book version not available",
	139:  "book is not available for download to Kindle (even with active package)",
	5143: "Legimi refused download (probably no downloads left in subscription period)",
}

type ErrorResponse struct {
	Type uint16
}

func (er ErrorResponse) Error() string {
	if message, ok := errorResponses[er.Type]; ok {
		return fmt.Sprintf(message)
	}
	return fmt.Sprintf("error response received: %d", er.Type)
}

func isErrorResponse(responseType uint16) bool {
	return responseType%2 == 1
}
