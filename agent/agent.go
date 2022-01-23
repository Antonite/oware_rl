package agent

import (
	"fmt"

	"github.com/Antonite/oware"
	"github.com/Antonite/oware_rl/storage"
)

type Agent struct {
	board   *oware.Board
	p1Moves map[string]bool
	p2Moves map[string]bool
	store   *storage.Storage
}

func new(store *storage.Storage) *Agent {
	b := oware.Initialize()
	return &Agent{
		board:   b,
		p1Moves: make(map[string]bool),
		p2Moves: make(map[string]bool),
		store:   store,
	}
}

func PlayForever(store *storage.Storage, id int) {
	for {
		a := new(store)
		a.play()
		fmt.Printf("game ended. worker: %v\n", id)
	}
}

func (a *Agent) play() {
	for a.board.Status == oware.InProgress {
		sroot := a.board.ToString()
		moves := a.board.GetValidMoves()
		if len(moves) == 0 {
			fmt.Printf("no valid moves: %s", sroot)
			return
		}

		var moveMap map[string]int
		// Get possible moves from history
		state, err := a.store.Get(sroot)
		if err != nil {
			// Entry doesn't exist
			moveMap = a.processPossibleMoves(moves)
			children := []string{}
			for k := range moveMap {
				children = append(children, k)
			}
			// Insert new record
			if err := a.store.Update(sroot, &storage.OwareState{Reward: 0, Children: children}); err != nil {
				panic(err)
			}
		} else if len(state.Children) == 0 {
			// Children are empty
			moveMap = a.processPossibleMoves(moves)
			children := []string{}
			for k := range moveMap {
				children = append(children, k)
			}
			// Add children
			if err := a.store.SafeAddChildren(sroot, children); err != nil {
				panic(err)
			}
		} else {
			// State and children exist, find out potential rewards
			moveMap = make(map[string]int, len(state.Children))
			for _, child := range state.Children {
				cstate, err := a.store.Get(child)
				if err != nil {
					fmt.Printf("failed to get child: %s\n", child)
					panic(err)
				}

				moveMap[child] = cstate.Reward
			}
		}

		// Decide on best move
		bestValue := 0
		bestMove := ""
		for k, v := range moveMap {
			// Ensure this move hasn't been played yet
			played := false
			if a.board.Player() == 0 {
				_, played = a.p1Moves[k]
			} else {
				_, played = a.p2Moves[k]
			}

			if !played && (bestMove == "" || v > bestValue) {
				bestMove = k
				bestValue = v
			}
		}

		// Can only repeat, must end game
		if bestMove == "" {
			a.board.ForceEndGame()
			fmt.Printf("forcefully ended game due to repetition: %s\n", sroot)
			continue
		}

		// Record for reward distribution
		if a.board.Player() == 0 {
			a.p1Moves[bestMove] = true
		} else {
			a.p2Moves[bestMove] = true
		}

		if bestValue > 0 {
			fmt.Printf("chose exp. reward: %v, %v\n", bestValue, bestMove)
		}

		// Convert the move
		nb, err := oware.NewS(bestMove)
		if err != nil {
			fmt.Printf("failed to convert board: %s\n", bestMove)
			panic(err)
		}

		a.board = nb
	}

	if a.board.Status == oware.Tie {
		// Skip reward distribution
		return
	} else if a.board.Status == oware.Player1Won {
		for m := range a.p1Moves {
			a.store.RewardChan <- m
		}
		for m := range a.p2Moves {
			a.store.PunishChan <- m
		}
	} else {
		for m := range a.p2Moves {
			a.store.RewardChan <- m
		}
		for m := range a.p1Moves {
			a.store.PunishChan <- m
		}
	}
}

func (a *Agent) processPossibleMoves(moves []int) map[string]int {
	childrenMap := make(map[string]int, len(moves))
	for _, m := range moves {
		cb, err := a.board.Move(m)
		if err != nil {
			fmt.Printf("failed to make move: %v\n", err)
			panic(err)
		}

		cbs := cb.ToString()
		var reward int
		if cb.Status == oware.InProgress || cb.Status == oware.Tie {
			reward = 0
		} else if cb.CurrentPlayerWon() {
			reward = 1000
		} else {
			reward = -1000
		}

		childrenMap[cbs] = reward
		state := &storage.OwareState{
			Reward: reward,
		}

		if err := a.store.Update(cbs, state); err != nil {
			fmt.Printf("failed to insert a child: %v\n", err)
			panic(err)
		}
	}

	return childrenMap
}
