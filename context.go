package kgen

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateContextKey generates a random string to be used as a context key
func GenerateContextKey() string {
	length := 10
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

var namespaceContextKey = GenerateContextKey()
