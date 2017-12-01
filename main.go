package main

import (
	"log"
	"math"
	"sync"
)

type simulationManager struct {
	running     bool
	initialized bool
	supervisor  supervisor
	waitgroup   sync.WaitGroup
	c           chan []byte
}

func newSimulationManager() *simulationManager {
	var mutex1, mutex2, mutex3, mutex4, mutex5 sync.RWMutex
	var wg sync.WaitGroup

	sv := supervisor{
		poolLock: mutex1,
		pool:     make(map[*node]struct{}),
		availabilityTableLock: mutex2,
		availabilityTable:     make(map[*node][]availabilityStatus),
		bwLock:                mutex3,
		bw:                    make(map[*node]float64),
		currentDownloadBwLock: mutex4,
		currentDownloadBw:     make(map[*node]float64),
		currentUploadBwLock:   mutex5,
		currentUploadBw:       make(map[*node]float64),
		file:                  newSegfile(12*MB, 10, 512*KB),
	}

	sm := simulationManager{
		running:     false,
		initialized: false,
		supervisor:  sv,
		waitgroup:   wg,
		c:           make(chan []byte),
	}

	return &sm
}

func (sm *simulationManager) initializeNodes() {
	var n *node
	for i := 0; i < 3; i++ {
		var ratio float64
		if ratio = 0.5; i == 0 {
			ratio = 1.0
		}
		n = newNode(&(sm.supervisor), 10*MB, 1-1/math.E, ratio)
		sm.supervisor.addNode(n)
	}
	sm.initialized = true
}

func (sm *simulationManager) start() {
	if sm.running {
		log.Println("Simulation already running!")
		return
	}
	if !sm.initialized {
		log.Println("Simulation not initialized!")
		return
	}

	sm.running = true
	log.Println("Starting simulation...")

	sm.supervisor.poolLock.RLock()
	for n := range sm.supervisor.pool {
		n.start(&sm.waitgroup)
	}
	sm.supervisor.poolLock.RUnlock()

	sm.waitgroup.Wait() // block
	log.Println("Simulation done!")
	sm.running = false
}

func main() {
	sm := newSimulationManager()
	sm.initializeNodes()
	sm.start()
}
