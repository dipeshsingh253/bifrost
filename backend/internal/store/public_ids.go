package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func newUUIDString() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Format as an RFC 4122 version 4 UUID without adding a dependency.
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(bytes[0:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:16]),
	), nil
}

func mustNewUUIDString() string {
	value, err := newUUIDString()
	if err != nil {
		panic(err)
	}
	return value
}
