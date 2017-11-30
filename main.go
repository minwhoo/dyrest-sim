package main

import (
	"fmt"
	"sync"
)

func initializeNodes(sv *supervisor) {
	var n *node
	for i := 0; i < 2; i++ {
		var ratio float64
		if ratio = 0.5; i == 0 {
			ratio = 1.0
		}
		n = newNode(sv, 1200*KB, 800*KB, 5, 10, ratio)
		sv.addNode(n)
	}
}

func startSimulation() {
	fmt.Println("Starting simulation...")
	var mutex1 sync.RWMutex
	var mutex2 sync.RWMutex
	var wg sync.WaitGroup
	segfile := newSegfile(12*MB, 10, 512*KB)

	sv := supervisor{
		make(map[*node]struct{}),
		mutex1,
		make(map[*node][]availabilityStatus),
		mutex2,
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
