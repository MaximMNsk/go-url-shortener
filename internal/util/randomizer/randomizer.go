package randomizer

import (
	"math/rand"
	"strconv"
)

func RandDigitalBytes(count int) (int, error) {
	min := 1
	maxVal := ``
	for i := 1; i <= count; i++ {
		maxVal += `9`
	}
	max, err := strconv.Atoi(maxVal)
	randInt := rand.Intn(max-min+1) + min
	return randInt, err
}
