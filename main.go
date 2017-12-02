package main

import (
	"log"
	"math"
	"sync"
)

var nodeIdx = 0

type simulationManager struct {
	running     bool
	initialized bool
	supervisor  supervisor
	waitgroup   sync.WaitGroup
	segfileInfo segfileInfo
}

func newSimulationManager() *simulationManager {
	var wg sync.WaitGroup

	sv := supervisor{
		poolLock:    sync.RWMutex{},
		pool:        make(map[*node]struct{}),
		bwRatioLock: sync.RWMutex{},
		bwRatio:     make(map[*node]float64),
		lg:          logger{make(chan []byte)},
	}

	sm := simulationManager{
		running:     false,
		initialized: false,
		supervisor:  sv,
		waitgroup:   wg,
		segfileInfo: newSegfileInfo(12*MB, 10, 512*KB),
	}

	return &sm
}

func (sm *simulationManager) initializeNodes() {
	nodeIdx = 0
	if sm.initialized {
		log.Println("SIM: ERROR Nodes already initialized!")
		return
	}
	var n *node
	for i := 0; i < 5; i++ {
		var ratio float64
		if ratio = 0.5; i == 0 {
			ratio = 1
		}
		n = newNode(&sm.segfileInfo, 10*MB, 1-1/math.E, ratio)
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
		n.start(&sm.supervisor, &sm.waitgroup)
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
	sm.initializeNodes()
	sm.start()
	//startWebInterface(sm)
}
