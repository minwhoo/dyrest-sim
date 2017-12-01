package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var nodeIdx = 0

type availabilityStatus int

const (
	statusNotAvailable availabilityStatus = iota
	statusPartiallyAvailable
	statusAvailable
)

type node struct {
	id                      int
	supervisor              *supervisor
	maxBw                   float64
	bwRatio                 float64
	connectedNodes          map[*node]struct{}
	dataChunkAvailability   []availabilityStatus
	parityChunkAvailability [][]availabilityStatus
	downc                   chan action
	complete                bool
}

func getRandomAvailability(ratio float64, numChunks int) []availabilityStatus {
	availability := make([]availabilityStatus, numChunks)
	rand.Seed(time.Now().UnixNano())

	for _, idx := range rand.Perm(numChunks)[:int(ratio*float64(numChunks))] {
		availability[idx] = statusAvailable
	}

	return availability
}

func newNode(sv *supervisor, maxBandwidth float64, bandwidthRatio float64, availabilityRatio float64) *node {
	n := node{
		id:                      nodeIdx,
		supervisor:              sv,
		maxBw:                   maxBandwidth,
		bwRatio:                 bandwidthRatio,
		connectedNodes:          make(map[*node]struct{}),
		dataChunkAvailability:   getRandomAvailability(availabilityRatio, sv.file.numDataChunks),
		parityChunkAvailability: [][]availabilityStatus{},
		downc:    make(chan action),
		complete: false,
	}
	if availabilityRatio == 1 {
		n.complete = true
	}
	nodeIdx++
	return &n
}

func (n *node) getMaxDownloadBw() float64 {
	return n.maxBw * n.bwRatio
}

func (n *node) getMaxUploadBw() float64 {
	return n.maxBw * (1 - n.bwRatio)
}

func (n *node) countAvailability() (na, pa, ya int) {
	na, pa, ya = 0, 0, 0
	for _, val := range n.dataChunkAvailability {
		switch val {
		case statusNotAvailable:
			na++
		case statusPartiallyAvailable:
			pa++
		case statusAvailable:
			ya++
		}
	}
	return
}

func (n *node) start(wg *sync.WaitGroup) {
	fmt.Println(n.id, "Starting node transfer")
	if !n.complete {
		wg.Add(1)
		go n.downloadLoop(wg)
	}
}

func (n *node) prepareTransfer(act action) {
	fmt.Println(n.id, ": node target", act.p.id, "selected")
	n.setAvailability(act.chk, statusPartiallyAvailable)
	n.connectedNodes[act.p] = struct{}{}
	n.supervisor.updateDownloadBw(n, +act.bw)
	n.supervisor.updateUploadBw(act.p, +act.bw)
}

func (n *node) transfer(act action) {
	transferTime := time.Duration(n.supervisor.file.chunkSize/act.bw*1000) * time.Millisecond
	time.Sleep(transferTime)
	n.downc <- act
}

func (n *node) transferDone(act action) {
	n.setAvailability(act.chk, statusAvailable)
	delete(n.connectedNodes, act.p)
	n.supervisor.updateDownloadBw(n, -act.bw)
	n.supervisor.updateUploadBw(act.p, -act.bw)
}

func (n *node) downloadLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	//resc := make(chan bool)
	var act action
	var na, pa int
	for {
		if na, pa, _ = n.countAvailability(); na == 0 {
			if pa == 0 {
				fmt.Println(n.id, ": Download complete!")
				n.complete = true
				break
			}
			goto block
		}

		act = n.supervisor.getOptimalAction(n, n.connectedNodes)
		if act.p == nil {
			if pa != 0 {
				goto block
			}
		} else {
			n.prepareTransfer(act)
			go n.transfer(act)
		}

		continue

	block:
		fmt.Println(n.id, "Blocked...")
		n.transferDone(<-n.downc)
	}
}

func (n *node) setAvailability(chk chunk, status availabilityStatus) {
	n.dataChunkAvailability[chk.idx] = status
	n.supervisor.updateAvailability(n, chk, status)
}
