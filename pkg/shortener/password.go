package shortener

import (
	"crypto/rand"
	"crypto/sha3"
)

const saltSize = 16

func hashPassword(password string) ([]byte, []byte, error) {
	salt := make([]byte, saltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, nil, err
	}

	hash := sha3.New512()
	hash.Write(salt)
	hash.Write([]byte(password))
	passwordHash := hash.Sum(nil)

	return salt, passwordHash, nil
}

func checkPasswordHash(
	password string,
	salt []byte,
	passwordHash []byte,
) (bool, error) {
	hash := sha3.New512()
	hash.Write(salt)
	hash.Write([]byte(password))
	hashBytes := hash.Sum(nil)
	return string(passwordHash) == string(hashBytes), nil
}
