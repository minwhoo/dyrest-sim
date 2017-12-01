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
}

func newSimulationManager() *simulationManager {
	var wg sync.WaitGroup

	sv := supervisor{
		poolLock: sync.RWMutex{},
		pool:     make(map[*node]struct{}),
		availabilityTableLock: sync.RWMutex{},
		availabilityTable:     make(map[*node][]availabilityStatus),
		bwLock:                sync.RWMutex{},
		bw:                    make(map[*node]float64),
		currentDownloadBwLock: sync.RWMutex{},
		currentDownloadBw:     make(map[*node]float64),
		currentUploadBwLock:   sync.RWMutex{},
		currentUploadBw:       make(map[*node]float64),
		file:                  newSegfile(12*MB, 10, 512*KB),
		lg:                    logger{make(chan []byte)},
	}

	sm := simulationManager{
		running:     false,
		initialized: false,
		supervisor:  sv,
		waitgroup:   wg,
	}

	return &sm
}

func (sm *simulationManager) initializeNodes() {
	if sm.initialized {
		log.Println("SIM: ERROR Nodes already initialized!")
		return
	}
	var n *node
	for i := 0; i < 100; i++ {
		var ratio float64
		if ratio = 0.0; i == 2 {
			ratio = 1.0
		}
		n = newNode(&(sm.supervisor), 10*MB, 1-1/math.E, ratio)
		sm.supervisor.addNode(n)
	}
	sm.initialized = true
	log.Println("SIM: Nodes initialized...")
}

func (sm *simulationManager) start() {
	if sm.running {
		log.Println("SIM: ERROR Simulation already running!")
		return
	}
	if !sm.initialized {
		log.Println("SIM: ERROR Simulation not initialized!")
		return
	}

	sm.running = true
	log.Println("SIM: Starting simulation...")

	sm.supervisor.poolLock.RLock()
	for n := range sm.supervisor.pool {
		n.start(&sm.waitgroup)
	}
	sm.supervisor.poolLock.RUnlock()

	sm.waitgroup.Wait() // block
	log.Println("SIM: Simulation done!")
	sm.running = false
}

func (sm *simulationManager) reset() {
	if sm.running {
		log.Println("SIM: ERROR Simulation not finished yet!")
		return
	}
	if !sm.initialized {
		log.Println("SIM: ERROR Nothing to reset!")
		return
	}

	pool := make(map[*node]struct{})

	// copy list of nodes to delete
	for n := range sm.supervisor.pool {
		pool[n] = struct{}{}
	}

	for n := range pool {
		sm.supervisor.removeNode(n)
	}

	sm.initialized = false
	log.Println("SIM: Supervisor reset!")
}

func main() {
	sm := newSimulationManager()
	startWebInterface(sm)
}
