package qdeepneuro

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/Antonite/oware"
	"gonum.org/v1/gonum/mat"
)

const (
	inputCount  int = 15
	weightCount int = 16
	outputCount int = 6
)

type network struct {
	mu            sync.Mutex
	layer1Weights *mat.Dense
	layer2Weights *mat.Dense
}

func newNetwork() *network {
	l1w := []float64{}
	l2w := []float64{}
	// Weights from input to hidden layer
	for i := 0; i < inputCount; i++ {
		for w := 0; w < weightCount; w++ {
			l1w = append(l1w, rand.Float64()-0.5)
		}
	}
	// Weights from hidden layer to output layer
	for i := 0; i < weightCount; i++ {
		for w := 0; w < outputCount; w++ {
			l2w = append(l2w, rand.Float64()-0.5)
		}
	}

	return &network{
		layer1Weights: mat.NewDense(inputCount, weightCount, l1w),
		layer2Weights: mat.NewDense(weightCount, outputCount, l2w),
	}
}

func (n *network) forward(state *oware.Board) (*oware.Board, int, error) {
	move, err := n.bestMove(state)
	if err != nil {
		return nil, move, err
	}

	nb, err := state.Move(move)
	if err != nil {
		return nil, move, err
	}

	return nb, move, nil
}

func (n *network) bestMove(state *oware.Board) (int, error) {
	// Valid moves
	valid := state.GetValidMoves()
	if len(valid) == 0 {
		return -1, errors.New("no valid moves found")
	}

	// Compute input layer
	inputVector := computeInputs(state)
	inputL := mat.NewDense(1, len(inputVector), inputVector)

	// Seed the board through the neural network
	_, _, outputL := n.internalNeuro(inputL)
	// Apply Softmax function
	max := softmax(outputL)

	// Compute move probability
	moveP := rand.Float64()
	// Select move
	valueSum := 0.0
	values := max.RawMatrix().Data
	if len(values) != 6 {
		return -1, errors.New("incorrect number of outputs computed")
	}
	for i, v := range valid {
		valueSum += values[i]
		if valueSum >= moveP {
			fmt.Printf("random value: %v\n", moveP)
			fmt.Printf("move value: %v\n", values[i])
			fmt.Printf("move taken: %v\n", v)
			return v, nil
		}
	}

	return -1, errors.New("failed to find a valid move")
}

func (n *network) internalNeuro(inputs *mat.Dense) (*mat.Dense, *mat.Dense, *mat.Dense) {
	// Apply W(i->1) weights
	rawHidden := make([]float64, weightCount)
	hiddenL := mat.NewDense(inputs.RawMatrix().Rows, weightCount, rawHidden)
	hiddenL.Mul(inputs, n.layer1Weights)

	hiddenInput := mat.NewDense(inputs.RawMatrix().Rows, weightCount, rawHidden)
	hiddenInput.Copy(hiddenL)
	// Apply Hidden Layer Activation functions
	applyActivations(hiddenL)

	// Apply W(1->o) weights
	rawOutput := make([]float64, outputCount)
	outputL := mat.NewDense(inputs.RawMatrix().Rows, outputCount, rawOutput)
	outputL.Mul(hiddenL, n.layer2Weights)
	return hiddenInput, hiddenL, outputL
}

func applyActivations(matrix *mat.Dense) {
	matrix.Apply(func(i int, j int, v float64) float64 {
		return leru(v)
	}, matrix)
}

func computeInputs(state *oware.Board) []float64 {
	inputs := []float64{float64(state.Player())}
	for _, s := range state.Scores() {
		inputs = append(inputs, float64(s))
	}
	for _, p := range state.Pits() {
		inputs = append(inputs, float64(p))
	}
	return inputs
}

func leru(x float64) float64 {
	if x < 0 {
		return 0
	}

	return x
}

func derLeru(x float64) float64 {
	if x < 0 {
		return 0
	}

	return 1
}

func softmax(matrix *mat.Dense) *mat.Dense {
	max := mat.Max(matrix)
	var sum float64
	matrix.Apply(func(i int, j int, v float64) float64 {
		n := v - max
		sum += math.Exp(n)
		return n
	}, matrix)

	resultMatrix := mat.NewDense(matrix.RawMatrix().Rows, matrix.RawMatrix().Cols, nil)
	resultMatrix.Apply(func(i int, j int, v float64) float64 {
		return math.Exp(v) / sum
	}, matrix)

	return resultMatrix
}
