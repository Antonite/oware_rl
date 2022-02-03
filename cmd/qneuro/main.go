package main

import (
	"fmt"

	"github.com/Antonite/oware_rl/qdeepneuro"
)

func main() {
	fmt.Println("starting oware deep q RL...")

	l := qdeepneuro.NewLeaner()
	l.Learn()
}
