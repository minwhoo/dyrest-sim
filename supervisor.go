package main

import "sync"

type supervisor struct {
	pool                  map[*node]struct{}
	poolLock              sync.RWMutex
	availabilityTable     map[*node][]availabilityStatus
	availabilityTableLock sync.RWMutex
	file                  segfile
}

type action struct {
	p   *node
	chk chunk
}

func (sv *supervisor) addNode(n *node) {
	sv.poolLock.Lock()
	defer sv.poolLock.Unlock()
	sv.availabilityTableLock.Lock()
	defer sv.availabilityTableLock.Unlock()

	sv.pool[n] = struct{}{}
	sv.availabilityTable[n] = n.dataChunkAvailability
}

func (sv *supervisor) removeNode(n *node) {
	sv.poolLock.Lock()
	defer sv.poolLock.Unlock()
	sv.availabilityTableLock.Lock()
	defer sv.availabilityTableLock.Unlock()

	delete(sv.pool, n)
	delete(sv.availabilityTable, n)
}

func (sv *supervisor) getOptimalAction(n *node, connectedNodes map[*node]struct{}) action {
	sv.poolLock.RLock()
	defer sv.poolLock.RUnlock()
	sv.availabilityTableLock.RLock()
	defer sv.availabilityTableLock.RUnlock()

	for p := range sv.pool {
		if _, ok := connectedNodes[p]; p != n && ok != true && p.complete {
			for i := 0; i < sv.file.numDataChunks; i++ {
				if sv.availabilityTable[n][i] == statusNotAvailable && sv.availabilityTable[p][i] == statusAvailable {
					return action{p, chunk{i, 0}}
				}
			}
		}
	}
	return action{nil, chunk{0, 0}}
}

func (sv *supervisor) updateAvailability(n *node, chk chunk, status availabilityStatus) {
	sv.availabilityTableLock.Lock()
	defer sv.availabilityTableLock.Unlock()
	sv.availabilityTable[n][chk.idx] = status
}
