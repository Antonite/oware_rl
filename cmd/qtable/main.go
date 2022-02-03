package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/Antonite/oware_rl/qtable"
	"github.com/Antonite/oware_rl/storage"
)

func main() {
	fmt.Println("starting oware RL...")

	store, err := storage.Init(1000)
	if err != nil {
		fmt.Println("failed to initialize storage")
		panic(err)
	}

	wg := sync.WaitGroup{}
	for w := 1; w <= 1000; w++ {
		wg.Add(1)
		time.Sleep(time.Millisecond * 200)
		go qtable.PlayForever(store, w)
	}

	wg.Wait()
}
