/*The MIT License (MIT)

Copyright (c) 2017 ActiveState Software Inc.

Written by Pete Garcin @rawktron

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.*/

package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"

	"strconv"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/text"
)

type synapse struct {
	inputNeuronIndex int
	weight           float64
	signal           float64
	x1, y1, x2, y2   float64
}

type neuron struct {
	x, y     float64
	output   float64
	synapses []synapse
	err      float64
	index    int
}

type layer struct {
	inputLayer    bool
	previousLayer *layer
	x, y          float64
	neurons       []neuron
}

type network struct {
	surface *imdraw.IMDraw
	text    *text.Text
	layers  []layer
}

type example struct {
	input  []float64
	output float64
}

func layer_left_margin(number_of_neurons int) float64 {
	return float64(44.0 + 110.0*(6.0-number_of_neurons)/2.0)

}

func sigmoid(x float64) float64 {
	return float64(1.0 / (1.0 + math.Exp(-x)))
}

func sigmoid_derivative(x float64) float64 {
	return x * (1 - x)
}

func (s *synapse) Draw(surface *imdraw.IMDraw) {
	if s.weight >= 0 {
		surface.Color = pixel.RGB(0, 1, 0)
	} else {
		surface.Color = pixel.RGB(1, 0, 0)
	}
	surface.Push(pixel.V(s.x1, s.y1), pixel.V(s.x2, s.y2))
	surface.Line(math.Abs(s.weight) + 1)
}

func (n *neuron) Draw(surface *imdraw.IMDraw, text *text.Text) {

	for _, s := range n.synapses {
		s.Draw(surface)
	}

	surface.Color = pixel.RGB(0.8, 0.8, 0.8)
	surface.Push(pixel.V(n.x, n.y))
	surface.Circle(40, 0)
	s := fmt.Sprintf("%.1f", n.output)
	text.Dot = pixel.V(n.x-20, n.y)
	text.WriteString(s)
}

func (n *neuron) Train(prevLayer *layer) layer {
	for _, s := range n.synapses {
		prevLayer.neurons[s.inputNeuronIndex].err += n.err * s.weight * sigmoid_derivative(n.output)
		s.weight += s.signal * n.err * sigmoid_derivative(n.output)
	}
	return *prevLayer
}

func (n *neuron) Think(prevLayer *layer) {
	activity := 0.0
	for i := range n.synapses {
		n.synapses[i].signal = prevLayer.neurons[n.synapses[i].inputNeuronIndex].output
		activity += n.synapses[i].weight * n.synapses[i].signal
		n.output = sigmoid(activity)
	}
}

func (l *layer) Think() {
	for i := range l.neurons {
		l.neurons[i].Think(l.previousLayer)
	}
}

func (l *layer) Draw(surface *imdraw.IMDraw, text *text.Text) {
	for _, neuron := range l.neurons {
		neuron.Draw(surface, text)
	}
}

func (n *network) NewNetwork(surface *imdraw.IMDraw, text *text.Text, requested []int) {
	n.surface = surface
	n.text = text
	n.layers = make([]layer, len(requested))

	f, err := os.Open("weights.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close() // this needs to be after the err check

	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		panic(err)
	}

	wi := 0

	for i := 0; i < len(requested); i++ {
		var newLayer layer

		if i == 0 {
			newLayer.inputLayer = true
			newLayer.previousLayer = nil
			newLayer.y = 600
		} else {
			newLayer.inputLayer = false
			newLayer.previousLayer = &n.layers[i-1]
			newLayer.y = newLayer.previousLayer.y - 120
		}

		x := layer_left_margin(requested[i])
		for j := 0; j < requested[i]; j++ {
			var neuron neuron
			neuron.x = x
			neuron.y = newLayer.y
			neuron.output = 0
			neuron.err = 0
			index := 0
			if newLayer.previousLayer != nil {
				for _, input := range newLayer.previousLayer.neurons {
					var synapse synapse
					synapse.inputNeuronIndex = index
					synapse.weight, _ = strconv.ParseFloat(lines[wi][0], 64)
					synapse.signal = 0
					synapse.x1 = x
					synapse.y1 = newLayer.y
					synapse.x2 = input.x
					synapse.y2 = input.y
					neuron.synapses = append(neuron.synapses, synapse)
					index++
					wi++
				}
			}

			newLayer.neurons = append(newLayer.neurons, neuron)
			x += 110
		}

		n.layers[i] = newLayer
	}
}

func (n *network) Train(data example) float64 {
	err := data.output - n.Think(data.input)
	n.ResetErrors()
	n.layers[len(n.layers)-1].neurons[0].err = err
	for i := len(n.layers) - 1; i > 0; i-- {
		for _, neuron := range n.layers[i].neurons {
			n.layers[i-1] = neuron.Train(&n.layers[i-1])
		}
	}
	return math.Abs(err)
}

func (n *network) ResetErrors() {
	for _, layer := range n.layers {
		for _, neuron := range layer.neurons {
			neuron.err = 0
		}
	}
}

func (n *network) Think(inputs []float64) float64 {
	for j, layer := range n.layers {
		if layer.inputLayer {
			for i := 0; i < len(inputs); i++ {
				n.layers[0].neurons[i].output = inputs[i]
			}
		} else {
			n.layers[j].Think()
		}
	}
	return n.layers[len(n.layers)-1].neurons[0].output
}

func (n *network) Draw() *imdraw.IMDraw {
	for _, layer := range n.layers {
		layer.Draw(n.surface, n.text)
	}

	return n.surface
}

func trainModel(nn *network) {
	file, err := os.Open("traindata.csv")
	if err != nil {
		// err is printable
		// elements passed are separated by space automatically
		fmt.Println("Error:", err)
		return
	}
	// automatically call Close() at the end of current method
	defer file.Close()
	//
	reader := csv.NewReader(file)
	// options are available at:
	// http://golang.org/src/pkg/encoding/csv/reader.go?s=3213:3671#L94
	reader.Comma = ','
	lineCount := 0
	for {
		// read just one record, but we could ReadAll() as well
		record, err := reader.Read()
		// end-of-file is fitted into err
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return
		}
		// record is an array of string so is directly printable
		fmt.Println("Record", lineCount, "is", record, "and has", len(record), "fields")
		// and we can iterate on top of that
		for i := 0; i < len(record); i++ {
			fmt.Println(" ", record[i])
		}
		fmt.Println()

		var trainExample example

		dx, _ := strconv.ParseFloat(record[0], 64)
		dy, _ := strconv.ParseFloat(record[1], 64)
		du, _ := strconv.ParseFloat(record[2], 64)
		dv, _ := strconv.ParseFloat(record[3], 64)

		trainExample.input = []float64{dx, dy, du, dv}
		trainExample.output, _ = strconv.ParseFloat(record[4], 64)

		fmt.Println(trainExample.input)

		nn.Train(trainExample)

		lineCount++
	}
}
