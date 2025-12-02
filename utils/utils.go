package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func DecodeBase64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}

func EncodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func Rand(len int) ([]byte, error) {
	salt := make([]byte, len)
	if n, err := rand.Read(salt); err != nil || n != len {
		return nil, fmt.Errorf("failed to generate salt: %v", err)
	}
	return salt, nil
}

func VerifyPassFormat(pw []byte) string {
	n := len(pw)
	if n < 12 || n > 64 {
		return "password length must be between 12 and 64 characters"
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSymbol := false

	runCount := 1
	for i := 0; i < n; i++ {
		b := pw[i]

		if b > 127 {
			return "only ASCII characters are allowed"
		}

		if b < 32 || b == 127 {
			return "control characters are not allowed"
		}

		if b == ' ' || b == '\t' {
			return "whitespace is not allowed"
		}

		switch {
		case b >= 'A' && b <= 'Z':
			hasUpper = true
		case b >= 'a' && b <= 'z':
			hasLower = true
		case b >= '0' && b <= '9':
			hasDigit = true
		default:
			hasSymbol = true
		}

		if i > 0 {
			if pw[i] == pw[i-1] {
				runCount++
				if runCount > 3 {
					return "a character repeats more than 3 times in a row"
				}
			} else {
				runCount = 1
			}
		}
	}

	if !hasUpper {
		return "must contain at least one uppercase letter"
	}
	if !hasLower {
		return "must contain at least one lowercase letter"
	}
	if !hasDigit {
		return "must contain at least one number"
	}
	if !hasSymbol {
		return "must contain at least one symbol"
	}

	return ""
}

func isLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// ABC-DEF-12
func VerifyBucKeyFormat(bucKey []byte) string {
	if len(bucKey) != 10 {
		return "invalid length"
	}

	for _, chr := range bucKey {
		if chr > 127 {
			return "only ASCII characters are allowed"
		}
	}

	for i := 0; i < 3; i++ {
		if !isLetter(bucKey[i]) {
			return "first three characters must be letters"
		}
	}

	if bucKey[3] != '-' {
		return "missing dash at position 4"
	}

	for i := 4; i < 7; i++ {
		if !isLetter(bucKey[i]) {
			return "middle three characters must be letters"
		}
	}

	if bucKey[7] != '-' {
		return "missing dash at position 8"
	}

	for i := 8; i < 10; i++ {
		if !isDigit(bucKey[i]) {
			return "last two characters must be digits"
		}
	}

	return ""
}
