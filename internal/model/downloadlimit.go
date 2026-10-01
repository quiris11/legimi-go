package model

import "math"

type DownloadLimit struct {
	Left uint32
	Max  uint32
}

// IsKnown reports whether limit was sent by Legimi (it is not, e.g. when package is not active)
func (dl DownloadLimit) IsKnown() bool {
	return dl.Max != math.MaxUint32
}
