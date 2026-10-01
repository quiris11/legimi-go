package book

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// Mobipocket file is a Palm database: header with list of record offsets,
// followed by records; last record is end of file marker.
const (
	mobiTypeOffset      = 60
	mobiRecordCountAt   = 76
	mobiRecordListStart = 78
	mobiRecordInfoSize  = 8
)

var (
	mobiType      = []byte("BOOKMOBI")
	mobiEndOfFile = []byte{0xe9, 0x8e, 0x0d, 0x0a}
)

// verifyMobi checks that downloaded file has expected size and complete Mobipocket structure
func verifyMobi(fileName string, expectedSize uint64) error {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	return verifyMobiData(data, expectedSize)
}

func verifyMobiData(data []byte, expectedSize uint64) error {
	if uint64(len(data)) != expectedSize {
		return fmt.Errorf("downloaded %d bytes, expected %d", len(data), expectedSize)
	}
	if len(data) < mobiRecordListStart || !bytes.Equal(data[mobiTypeOffset:mobiTypeOffset+len(mobiType)], mobiType) {
		return fmt.Errorf("downloaded file is not a Mobipocket book")
	}
	recordCount := int(binary.BigEndian.Uint16(data[mobiRecordCountAt:]))
	recordListEnd := mobiRecordListStart + recordCount*mobiRecordInfoSize
	if recordCount == 0 || len(data) < recordListEnd {
		return fmt.Errorf("downloaded book is corrupted: invalid record list")
	}
	previousOffset := uint32(recordListEnd)
	for i := 0; i < recordCount; i++ {
		offset := binary.BigEndian.Uint32(data[mobiRecordListStart+i*mobiRecordInfoSize:])
		if offset < previousOffset || uint64(offset) >= uint64(len(data)) {
			return fmt.Errorf("downloaded book is corrupted: invalid offset of record %d", i)
		}
		previousOffset = offset
	}
	if !bytes.HasSuffix(data, mobiEndOfFile) {
		return fmt.Errorf("downloaded book is incomplete: end of file marker missing")
	}
	return nil
}
