package tools

import (
	"math/rand"
	"time"
)

func RandomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}

	return string(bytes)
}

func randInt(min, max int) int {
	generator := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + generator.Intn(max-min)
}
