package main

import (
	"fmt"
)

func initializeNodes(pool *[]*node, file *segfile) {
	for i := 0; i < 10; i++ {
		var ratio float64
		if ratio = 0.5; i == 0 {
			ratio = 1.0
		}
		newNode(pool, file, 100, 10, 5, 10, ratio)
	}
}

func startSimulation() {
	fmt.Println("Starting simulation...")

	segfile := newSegfile(12*MB, 10, 512*KB)

	nodePool := []*node{}
	initializeNodes(&nodePool, &segfile)

	/* TESTGROUND */
	c := make(chan bool)
	nodePool[0].startSeed()
	nodePool[1].startDownload(c)
	nodePool[2].startDownload(c)
	nodePool[3].startDownload(c)
	<-c //block
	fmt.Println("Finished")
	/* 	   END    */

	fmt.Println("Simulation done!")
}

func main() {
	startSimulation()
}
