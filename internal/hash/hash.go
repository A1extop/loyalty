package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// интерфейс + структура !
// Хеширует пароль

func HashPassword(password string, key string) (string, error) {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(password))
	dst := h.Sum(nil)
	hashHex := hex.EncodeToString(dst)
	return hashHex, nil
}
