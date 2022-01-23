package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	for w := 1; w <= 200; w++ {
		a := agent.New(store)
		go a.Play()
	}

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	store.Close()
}
