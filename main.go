package main

import (
	"fmt"
)

func initializeNodes(pool *[]*node, file *segfile) {
	for i := 0; i < 5; i++ {
		newNode(pool, file, 100, 10, 10, 10)
	}
}

func startSimulation() {
	fmt.Println("Starting simulation...")
	segfile := newSegfile(12*1024, 10, 512)
	nodePool := []*node{}
	initializeNodes(&nodePool, &segfile)
	fmt.Println("Simulation done!")
}

func main() {
	startSimulation()
}
