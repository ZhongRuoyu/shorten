package shortener

import (
	"crypto/rand"
	"math/big"
	"regexp"
)

var alphabet = []rune(
	"0123456789" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz")
var codeRegex = regexp.MustCompile("^[-_0-9A-Za-z]+$")

func generateCode(length int) (string, error) {
	code := make([]rune, length)
	for i := range code {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		code[i] = alphabet[index.Int64()]
	}
	return string(code), nil
}

func isValidCode(code string) bool {
	return codeRegex.MatchString(code)
}
