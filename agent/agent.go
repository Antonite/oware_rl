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

func New(store *storage.Storage) *Agent {
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
		a := New(store)
		a.Play()
	}
}

func (a *Agent) Board() *oware.Board {
	return a.board
}

func (a *Agent) SetBoard(board *oware.Board) {
	a.board = board
}

func (a *Agent) Play() {
	for a.board.Status == oware.InProgress {
		sroot := a.board.ToString()
		moves := a.board.GetValidMoves()
		if len(moves) == 0 {
			fmt.Printf("no valid moves: %s", sroot)
			return
		}

		// Get possible moves with reward values
		moveMap := a.ExploreCurrentMoves(moves, sroot)

		// Decide on best move
		bestValue := 0
		bestMove := ""
		for k, v := range moveMap {
			// Ensure this move hasn't been played yet
			played := a.MovePlayed(k)
			if !played && (bestMove == "" || v > bestValue) {
				bestMove = k
				bestValue = v
			}
		}

		// Can only repeat, must end game
		if bestMove == "" {
			a.board.ForceEndGame()
			continue
		}

		// Record for reward distribution
		a.RecordMove(bestMove)

		// Convert the move
		nb, err := oware.NewS(bestMove)
		if err != nil {
			fmt.Printf("failed to convert board: %s\n", bestMove)
			return
		}

		a.board = nb
	}

	a.DistributeAwards()
}

func (a *Agent) DistributeAwards() {
	if a.board.Status == oware.Tie {
		fmt.Printf("skipped distribution %s\n", a.board.ToString())
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

func (a *Agent) RecordMove(move string) {
	if a.board.Player() == 0 {
		a.p1Moves[move] = true
	} else {
		a.p2Moves[move] = true
	}
}

func (a *Agent) MovePlayed(move string) bool {
	played := false
	if a.board.Player() == 0 {
		_, played = a.p1Moves[move]
	} else {
		_, played = a.p2Moves[move]
	}
	return played
}

func (a *Agent) ExploreCurrentMoves(moves []int, sroot string) map[string]int {
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
		a.store.Insert(sroot, &storage.OwareState{Reward: 0, Children: children})
	} else if len(state.Children) == 0 {
		// Children are empty
		moveMap = a.processPossibleMoves(moves)
		children := []string{}
		for k := range moveMap {
			children = append(children, k)
		}
		// Add children
		if err := a.store.SafeAddChildren(sroot, children); err != nil {
			fmt.Printf("failed to save children: %s\n", sroot)
		}
	} else {
		// State and children exist, find out potential rewards
		moveMap = make(map[string]int, len(state.Children))
		for _, child := range state.Children {
			cstate, err := a.store.Get(child)
			reward := 0
			if err != nil {
				fmt.Printf("failed to get child: %s\n", child)
			} else {
				reward = cstate.Reward
			}

			moveMap[child] = reward
		}
	}

	return moveMap
}

func (a *Agent) processPossibleMoves(moves []int) map[string]int {
	childrenMap := make(map[string]int, len(moves))
	for _, m := range moves {
		cb, err := a.board.Move(m)
		if err != nil {
			fmt.Printf("failed to make move: %v\n", err)
			continue
		}

		cbs := cb.ToString()
		var reward int
		if cb.Status == oware.InProgress {
			player := (cb.Player() + 1) % 2
			reward = cb.Scores()[player]
		} else if cb.Status == oware.Tie {
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

		a.store.Insert(cbs, state)
	}

	return childrenMap
}
