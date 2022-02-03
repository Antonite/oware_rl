package qdeepneuro

import (
	"gonum.org/v1/gonum/mat"
)

const (
	learners     int     = 1000
	learningRate float64 = 0.5
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
		if len(acts) < 1 {
			acts = append(acts, act)
		} else {
			// Input layer
			inputVector := []float64{}
			experimentalInputVector := []float64{}
			for _, a := range acts {
				inputVector = append(inputVector, computeInputs(a.current)...)
				experimentalInputVector = append(experimentalInputVector, computeInputs(a.new)...)
			}

			inputL := mat.NewDense(inputCount, len(acts), inputVector)
			eInputL := mat.NewDense(inputCount, len(acts), experimentalInputVector)

			l.network.mu.Lock() // Lock weights
			hiddenI, hiddenL, outputL := l.network.internalNeuro(inputL)
			_, _, eOutputL := l.network.internalNeuro(eInputL)

			// Sum all pit values to get overall state value
			outputCollapsedV := []float64{}
			eOutputCollapsedV := []float64{}
			for r := 0; r < outputL.RawMatrix().Rows; r++ {
				sum := 0.0
				eSum := 0.0
				for c := 0; c < outputL.RawMatrix().Cols; c++ {
					sum += outputL.At(r, c)
					eSum += eOutputL.At(r, c)
				}
				outputCollapsedV = append(outputCollapsedV, sum)
				eOutputCollapsedV = append(eOutputCollapsedV, eSum)
			}
			outputCollapsed := mat.NewDense(outputL.RawMatrix().Rows, 1, outputCollapsedV)
			eOutputCollapsed := mat.NewDense(outputL.RawMatrix().Rows, 1, eOutputCollapsedV)

			// Find error rate
			eOutputCollapsed.Sub(eOutputCollapsed, outputCollapsed)

			// Change output weights
			chngOutV := make([]float64, weightCount*outputCount)
			chngOut := mat.NewDense(weightCount, outputCount, chngOutV)
			chngOut.Mul(eOutputL, hiddenL)
			chngOut.Apply(func(i, j int, v float64) float64 { return -learningRate * v }, chngOut)

			// Apply weight change
			l.network.layer2Weights.Add(l.network.layer2Weights, chngOut)

			// Change L1 weights
			chngL1V := make([]float64, inputCount*weightCount)
			chngL1 := mat.NewDense(inputCount, weightCount, chngL1V)
			chngL1.Mul(eOutputL, hiddenL)

			l.network.mu.Unlock() // Unlock weights
		}
	}
}
