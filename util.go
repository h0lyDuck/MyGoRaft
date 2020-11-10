package main

import (
	"math/rand"
	"time"
)

func randRange(min, max int64) int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63n(max-min) + min
}

func getCurrentTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
