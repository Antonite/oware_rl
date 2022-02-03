package qdeepneuro

import "github.com/Antonite/oware"

type memory struct {
	actions chan *action
}

type action struct {
	current *oware.Board
	new     *oware.Board
	move    int
}

func newMemory() *memory {
	return &memory{
		actions: make(chan *action, 100000),
	}
}
