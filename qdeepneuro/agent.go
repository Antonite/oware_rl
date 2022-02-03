package qdeepneuro

import (
	"fmt"

	"github.com/Antonite/oware"
)

type agent struct {
	board   *oware.Board
	memory  *memory
	network *network
}

func newAgent(network *network, memory *memory) *agent {
	b := oware.Initialize()
	return &agent{
		board:   b,
		memory:  memory,
		network: network,
	}
}

func (a *agent) play() {
	fmt.Printf("starting board: %v\n", a.board)
	board, move, err := a.network.forward(a.board)
	if err != nil {
		fmt.Printf("failed to forward. err: %v\n", err)
		return
	}

	fmt.Printf("move 1 board: %v\n", board)
	if board.Status != oware.InProgress {
		// Award right away
		return
	}

	board, move, err = a.network.forward(board)
	if err != nil {
		fmt.Printf("failed to forward. err: %v\n", err)
		return
	}

	fmt.Printf("move 2 board: %v\n", board)

	// Record for future learning
	a.memory.actions <- &action{a.board, board, move}
}
