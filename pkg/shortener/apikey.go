package shortener

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const apiKeySize = 32

func GenerateApiKey() (string, error) {
	b := make([]byte, apiKeySize)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func HashApiKey(key string) (string, error) {
	b, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
