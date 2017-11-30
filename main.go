package main

import (
	"fmt"
	"math"
	"sync"
)

func initializeNodes(sv *supervisor) {
	var n *node
	for i := 0; i < 4; i++ {
		var ratio float64
		if ratio = 0.5; i == 0 {
			ratio = 1.0
		}
		n = newNode(sv, 10*MB, 1-1/math.E, ratio)
		sv.addNode(n)
	}
}

func startSimulation() {
	fmt.Println("Starting simulation...")
	var mutex1, mutex2, mutex3 sync.RWMutex
	var wg sync.WaitGroup
	segfile := newSegfile(12*MB, 10, 512*KB)

	sv := supervisor{
		mutex1,
		make(map[*node]struct{}),
		mutex2,
		make(map[*node][]availabilityStatus),
		mutex3,
		make(map[*node]float64),
		segfile,
	}

	initializeNodes(&sv)

	/* TESTGROUND */
	sv.poolLock.RLock()
	for n := range sv.pool {
		n.start(&wg)
	}
	sv.poolLock.RUnlock()
	/* 	   END    */

	wg.Wait() // block
	fmt.Println("Simulation done!")
}

func main() {
	startSimulation()
}
