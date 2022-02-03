package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Antonite/oware"
	"github.com/Antonite/oware_rl/qtable"
	"github.com/Antonite/oware_rl/storage"
)

func main() {
	var player = flag.Int("player", 0, "[0,1]")
	flag.Parse()
	if *player != 0 && *player != 1 {
		flag.Usage()
		return
	}

	fmt.Printf("starting oware client for player: %v\n", *player)

	store, err := storage.Init(50)
	if err != nil {
		fmt.Println("failed to initialize storage")
		panic(err)
	}

	a := qtable.New(store)
	for a.Board().Status == oware.InProgress {
		sroot := a.Board().ToString()
		moves := a.Board().GetValidMoves()

		// Get possible moves with reward values
		moveMap := a.ExploreCurrentMoves(moves, sroot)
		pitMap := make(map[string]int)

		fmt.Println("-------------------------------------------")
		fmt.Println("-------------------------------------------")
		fmt.Printf("Board state: %v\n", a.Board())
		fmt.Println("Move options:")
		for _, m := range moves {
			nb, err := a.Board().Move(m)
			if err != nil {
				continue
			}

			nbs := nb.ToString()

			reward, ok := moveMap[nbs]
			if !ok {
				fmt.Printf("couldn't find move in storage: %s, for key: %s\n", nbs, sroot)
				return
			}

			pitMap[nbs] = m

			fmt.Printf("Pit: %v Outcome: %s Reward: %v\n", m, nbs, reward)
		}

		// Has this moved been played yet?
		played := false
		bestMove := ""

		if a.Board().Player() == *player {
			// Player's turn
			fmt.Println("Your turn. Waiting for move selection...")

			// wait for input
			input := bufio.NewScanner(os.Stdin)
			waitforinput := true
			for waitforinput {
				input.Scan()
				playerMove, err := strconv.Atoi(input.Text())
				if err != nil {
					fmt.Println("bad input, try again")
					continue
				}

				nb, err := a.Board().Move(playerMove)
				if err != nil {
					fmt.Println("bad input, try again")
					continue
				}

				bestMove = nb.ToString()
				waitforinput = false
				fmt.Printf("You chose pit: %v\n", playerMove)
			}

			// Ensure this move hasn't been played yet
			played = a.MovePlayed(bestMove)
		} else {
			// AI's turn
			// Decide on best move
			fmt.Println("AI's turn. Waiting for move selection...")
			bestValue := 0
			for k, v := range moveMap {
				// Ensure this move hasn't been played yet
				played = a.MovePlayed(k)
				if !played && (bestMove == "" || v > bestValue) {
					bestMove = k
					bestValue = v
				}
			}

			if bestMove == "" {
				played = true
			}

			pit, ok := pitMap[bestMove]
			if !ok {
				fmt.Printf("AI chose: end game")
			} else {
				fmt.Printf("AI chose: %v\n", pit)
			}
		}

		// Repeating, must end game
		if played {
			a.Board().ForceEndGame()
			fmt.Printf("forcefully ended game due to repetition: %s\n", sroot)
			continue
		}

		// Record for reward distribution
		a.RecordMove(bestMove)

		// Convert the move
		nb, err := oware.NewS(bestMove)
		if err != nil {
			fmt.Printf("failed to convert board: %s\n", bestMove)
			panic(err)
		}

		a.SetBoard(nb)
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("-------------------------------------------")
	fmt.Println("-------------------------------------------")
	fmt.Println("Game ended. Distributing awards")
	fmt.Println(a.Board())

	a.DistributeAwards()

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	store.Close()
}
