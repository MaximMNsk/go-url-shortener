package randomizer

import (
	"math/rand"
	"strconv"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func RandDigitalBytes(capacity int) (int, error) {
	if capacity >= 12 {
		return rand.Intn(999999999999), nil
	}
	x := "9"
	for i := 0; i < capacity; i++ {
		x += "9"
	}
	y, err := strconv.Atoi(x)
	if err != nil {
		return 0, err
	}
	return rand.Intn(y), nil
}
