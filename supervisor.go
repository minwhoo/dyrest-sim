package main

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type supervisor struct {
	poolLock              sync.RWMutex
	pool                  map[*node]struct{}
	availabilityTableLock sync.RWMutex
	availabilityTable     map[*node][]availabilityStatus
	bwLock                sync.RWMutex
	bw                    map[*node]float64
	currentDownloadBwLock sync.RWMutex
	currentDownloadBw     map[*node]float64
	currentUploadBwLock   sync.RWMutex
	currentUploadBw       map[*node]float64
	file                  segfile
	lg                    logger
}

type action struct {
	p   *node
	chk chunk
	bw  float64
}

func (sv *supervisor) addNode(n *node) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	sv.poolLock.Lock()
	sv.pool[n] = struct{}{}
	sv.poolLock.Unlock()

	sv.availabilityTableLock.Lock()
	sv.availabilityTable[n] = n.dataChunkAvailability
	sv.availabilityTableLock.Unlock()

	sv.bwLock.Lock()
	chunkDownloadTime := math.Max(0.1, random.NormFloat64()*0.3+0.6)
	sv.bw[n] = sv.file.chunkSize / chunkDownloadTime
	sv.bwLock.Unlock()

	sv.currentDownloadBwLock.Lock()
	sv.currentDownloadBw[n] = 0
	sv.currentDownloadBwLock.Unlock()

	sv.currentUploadBwLock.Lock()
	sv.currentUploadBw[n] = 0
	sv.currentUploadBwLock.Unlock()

	sv.lg.logNodeAdded(n)
}

func (sv *supervisor) removeNode(n *node) {
	sv.poolLock.Lock()
	delete(sv.pool, n)
	sv.poolLock.Unlock()

	sv.availabilityTableLock.Lock()
	delete(sv.availabilityTable, n)
	sv.availabilityTableLock.Unlock()

	sv.bwLock.Lock()
	delete(sv.bw, n)
	sv.bwLock.Unlock()

	sv.currentDownloadBwLock.Lock()
	delete(sv.currentDownloadBw, n)
	sv.currentDownloadBwLock.Unlock()

	sv.currentUploadBwLock.Lock()
	delete(sv.currentUploadBw, n)
	sv.currentUploadBwLock.Unlock()
}

func (sv *supervisor) getOptimalAction(n *node, connectedNodes map[*node]struct{}) action {
	sv.poolLock.RLock()
	defer sv.poolLock.RUnlock()
	sv.availabilityTableLock.RLock()
	defer sv.availabilityTableLock.RUnlock()
	sv.bwLock.RLock()
	defer sv.bwLock.RUnlock()
	sv.currentDownloadBwLock.RLock()
	defer sv.currentDownloadBwLock.RUnlock()
	sv.currentUploadBwLock.RLock()
	defer sv.currentUploadBwLock.RUnlock()

	for p := range sv.pool {
		if _, ok := connectedNodes[p]; p != n && ok != true {
			for i := 0; i < sv.file.numDataChunks; i++ {
				if sv.availabilityTable[n][i] == statusNotAvailable && sv.availabilityTable[p][i] == statusAvailable {
					if sv.bw[p] <= p.getMaxUploadBw()-sv.currentUploadBw[p] && sv.bw[p] <= n.getMaxDownloadBw()-sv.currentDownloadBw[n] {
						return action{p, chunk{i, 0}, sv.bw[p]}
					}
				}
			}
		}
	}

	return action{nil, chunk{0, 0}, 0}
}

func (sv *supervisor) updateDownloadBw(n *node, deltaBw float64) {
	sv.currentDownloadBwLock.Lock()
	sv.currentDownloadBw[n] += deltaBw
	sv.currentDownloadBwLock.Unlock()
}

func (sv *supervisor) updateUploadBw(n *node, deltaBw float64) {
	sv.currentUploadBwLock.Lock()
	sv.currentUploadBw[n] += deltaBw
	sv.currentUploadBwLock.Unlock()
}

func (sv *supervisor) updateAvailability(n *node, chk chunk, status availabilityStatus) {
	sv.availabilityTableLock.Lock()
	sv.availabilityTable[n][chk.idx] = status
	sv.availabilityTableLock.Unlock()
	sv.lg.logNodeAvailabilityUpdated(n.id, chk, status)
}
