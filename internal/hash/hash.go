package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func HashPassword(password string, key string) (string, error) {
	pas, err := json.Marshal(password)
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(pas)
	dst := h.Sum(nil)
	hashHex := hex.EncodeToString(dst)
	return hashHex, nil
}
