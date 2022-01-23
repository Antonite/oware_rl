package main

import (
	"fmt"

	"github.com/Antonite/oware_rl/agent"
	"github.com/Antonite/oware_rl/storage"
)

func main() {
	fmt.Println("starting oware RL...")

	store, err := storage.Init()
	if err != nil {
		fmt.Println("failed to initialize storage")
		panic(err)
	}

	a := agent.New(store)
	a.Play()
}
