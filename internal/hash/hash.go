package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
)

// Вроде отвязался от конкретной реализации

var key = []byte("secretKey")

type Hasher interface {
	New() hash.Hash
}
type SHA256Hasher struct{}

func (s SHA256Hasher) New() hash.Hash {
	return sha256.New()
}
func NewSHA256Hasher() Hasher {
	return &SHA256Hasher{}
}

// Хеширует пароль
func HashPassword(password string, hasher Hasher) (string, error) {
	if hasher == nil {
		return "", errors.New("hash function is nil")
	}

	h := hmac.New(hasher.New, key)
	_, err := h.Write([]byte(password))
	if err != nil {
		return "", err
	}
	dst := h.Sum(nil)
	hashHex := hex.EncodeToString(dst)
	return hashHex, nil
}
