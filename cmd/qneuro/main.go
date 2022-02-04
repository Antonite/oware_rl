package main

import (
	"fmt"
	"time"

	"github.com/Antonite/oware_rl/qdeepneuro"
)

func main() {
	fmt.Println("starting oware deep q RL...")

	l := qdeepneuro.NewLeaner()
	l.Learn()

	time.Sleep(time.Minute)
}
