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

func VerifyPassFormat(pwd []byte) string {
	// TODO:
	return ""
}

func VerifyBucKeyFormat(bucKey []byte) string {
	// TODO:
	return ""
}
