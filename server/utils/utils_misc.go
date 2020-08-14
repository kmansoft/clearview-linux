package utils

import "math/rand"

func RandomChoice(list []string) string {
	n := rand.Intn(len(list))
	return list[n]
}
