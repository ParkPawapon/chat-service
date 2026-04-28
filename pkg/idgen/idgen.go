package idgen

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

func NewUUID() string {
	return uuid.NewString()
}

func HashIdentifier(identifier string) string {
	sum := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(sum[:])
}
