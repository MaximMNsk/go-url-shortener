// Package randomizer - генератор случайных чисел заданной длины.
package randomizer

import (
	"math/rand"
	"strconv"
)

// RandDigitalBytes - возвращает число заданной длины и ошибку.
func RandDigitalBytes(count int) (int, error) {
	minimum := 1
	maxVal := ``

	for i := 1; i <= count; i++ {
		maxVal += `9`
	}

	maximum, err := strconv.Atoi(maxVal)
	randInt := rand.Intn(maximum-minimum+1) + minimum

	return randInt, err
}
