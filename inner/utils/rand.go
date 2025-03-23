package utils

import (
	"fmt"
	"math/rand"
)

func GetKey(n int) []byte {
	return []byte(fmt.Sprintf("key-%d", n))
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GetValue(length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}
