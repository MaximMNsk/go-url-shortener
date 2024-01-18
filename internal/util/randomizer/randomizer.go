package randomizer

import (
	"math/rand"
	"strconv"
)

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
