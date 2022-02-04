package qdeepneuro

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

const (
	learners     int     = 1000
	learningRate float64 = 1
)

type Learner struct {
	network *network
	memory  *memory
}

func NewLeaner() *Learner {
	l := &Learner{
		network: newNetwork(),
		memory:  newMemory(),
	}

	for w := 1; w <= learners; w++ {
		go l.remember()
	}

	return l
}

func (l *Learner) Learn() {
	a := newAgent(l.network, l.memory)
	a.play()
}

func (l *Learner) remember() {
	acts := []*action{}
	for act := range l.memory.actions {
		acts = append(acts, act)
		if len(acts) < 1 {
			continue
		}

		// Input layer
		inputVector := []float64{}
		experimentalInputVector := []float64{}
		for _, a := range acts {
			inputVector = append(inputVector, computeInputs(a.current)...)
			experimentalInputVector = append(experimentalInputVector, computeInputs(a.new)...)
		}

		inputL := mat.NewDense(len(acts), inputCount, inputVector)
		eInputL := mat.NewDense(len(acts), inputCount, experimentalInputVector)

		l.network.mu.Lock() // Lock weights
		_, hiddenL, outputL := l.network.internalNeuro(inputL)
		_, _, eOutputL := l.network.internalNeuro(eInputL)

		// Find error rate
		fmt.Printf("CURRENT: %v\n", outputL)
		fmt.Printf("EXPERIMENTAL: %v\n", eOutputL)
		outputL.Sub(eOutputL, outputL)
		fmt.Printf("ERROR RATE: %v\n", outputL)

		// Change output weights
		chngOutV := make([]float64, weightCount*outputCount)
		chngOut := mat.NewDense(weightCount, outputCount, chngOutV)
		// fmt.Printf("chngOut layer: %v\n", chngOut)
		// fmt.Printf("hiddenL layer: %v\n", hiddenL)
		// fmt.Printf("outputL layer: %v\n", outputL)
		chngOut.Mul(hiddenL.T(), outputL)
		chngOut.Apply(func(i, j int, v float64) float64 { return -learningRate * v }, chngOut)
		// Apply weight change
		// fmt.Printf("chngOut layer: %v\n", chngOut)
		fmt.Printf("weights before: %v\n", l.network.layer2Weights)
		l.network.layer2Weights.Add(l.network.layer2Weights, chngOut)
		fmt.Printf("weights after: %v\n", l.network.layer2Weights)

		// // Change L1 weights
		// chngOut2 := mat.NewDense(len(acts), weightCount, chngOutV)
		// chngOut2.Mul(outputL, l.network.layer2Weights.T())
		// // Derivative of hidden layer with respect to its input
		// applyDerLeru(hiddenI)

		// chngL1V := make([]float64, inputCount*weightCount)
		// chngL1 := mat.NewDense(inputCount, weightCount, chngL1V)
		// // fmt.Printf("hiddenI layer: %v\n", hiddenI)
		// hiddenI.MulElem(hiddenI, chngOut2)
		// chngL1.Mul(inputL.T(), hiddenI)
		// // fmt.Printf("chngL1 layer: %v\n", chngL1)
		// chngL1.Apply(func(i, j int, v float64) float64 { return -learningRate * v }, chngL1)
		// // fmt.Printf("chngL1 layer: %v\n", chngL1)
		// // fmt.Printf("weights before: %v\n", l.network.layer1Weights)
		// l.network.layer1Weights.Add(l.network.layer1Weights, chngL1)
		// // fmt.Printf("weights after: %v\n", l.network.layer1Weights)

		l.network.mu.Unlock() // Unlock weights
	}
}
