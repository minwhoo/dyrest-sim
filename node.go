package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type availabilityStatus int

const (
	statusNotAvailable availabilityStatus = iota
	statusPartiallyAvailable
	statusAvailable
)

type bandwidth struct {
	sync.RWMutex
	value float64
}

func (bw *bandwidth) update(deltaBw float64) {
	bw.Lock()
	defer bw.Unlock()
	//fmt.Printf("Updating bandwidth..\n")
	bw.value += deltaBw
}

func (bw *bandwidth) get() float64 {
	bw.RLock()
	defer bw.RUnlock()
	return bw.value
}

type node struct {
	id                int
	sf                segfile
	currentDownloadBw bandwidth
	currentUploadBw   bandwidth
	maxBw             float64
	maxBwRatio        float64
	connectedNodes    map[*node]struct{}
	c                 chan transferResult
	complete          bool
	simTime           float64
}

type transferResult struct {
	act        action
	finishTime float64
}

func newNode(sfi *segfileInfo, maxBandwidth float64, bandwidthRatio float64, availabilityRatio float64) *node {
	n := node{
		id:                nodeIdx,
		sf:                newSegfile(sfi),
		currentDownloadBw: bandwidth{sync.RWMutex{}, 0},
		currentUploadBw:   bandwidth{sync.RWMutex{}, 0},
		maxBw:             maxBandwidth,
		maxBwRatio:        bandwidthRatio,
		connectedNodes:    make(map[*node]struct{}),
		c:                 make(chan transferResult),
		complete:          false,
		simTime:           0,
	}
	n.getRandomAvailability(availabilityRatio)
	if availabilityRatio == 1 {
		n.complete = true
	}
	nodeIdx++
	return &n
}

func (n *node) getRandomAvailability(ratio float64) {
	rand.Seed(time.Now().UnixNano())

	for _, idx := range rand.Perm(n.sf.numDataChunks)[:int(ratio*float64(n.sf.numDataChunks))] {
		chkId := chunkId{idx / n.sf.segmentSize, 0, idx % n.sf.segmentSize}
		n.sf.setChunk(chkId, statusAvailable)
	}
}

func (n *node) getMaxDownloadBw() float64 {
	return n.maxBw * n.maxBwRatio
}

func (n *node) getMaxUploadBw() float64 {
	return n.maxBw * (1 - n.maxBwRatio)
}

func (n *node) start(sv *supervisor, wg *sync.WaitGroup) {
	fmt.Println(n.id, ": ====== Starting node transfer ======")
	if !n.complete {
		wg.Add(1)
		go n.downloadLoop(sv, wg)
	}
}

func (n *node) downloadLoop(sv *supervisor, wg *sync.WaitGroup) {
	defer wg.Done()
	var act action
	for {
		if n.sf.plannedComplete() {
			if !n.sf.transferInProgress() {
				fmt.Println(n.id, ": Download complete!, total time taken: ", n.simTime)
				n.complete = true
				break
			}
			goto block
		}

		act = sv.getOptimalAction(n, n.connectedNodes)
		if act.p == nil {
			if n.sf.transferInProgress() {
				goto block
			}
		} else {
			fmt.Printf("%v <---(s: %v,c:%v,r:%v)---- %v : %.2f MB/s\n", n.id, act.chkId.sIdx, act.chkId.cIdx, act.chkId.rIdx, act.p.id, act.bw/MB)
			n.prepareTransfer(act)
			go n.transfer(act)
		}

		continue

	block:
		//fmt.Println(n.id, "Blocked...")
		result := <-n.c
		n.simTime = math.Max(n.simTime, result.finishTime)
		n.transferDone(result.act)
		//fmt.Printf("%v >> ", n.id)
		//n.sf.showSegments()
	}
}

func (n *node) prepareTransfer(act action) {
	n.sf.setChunk(act.chkId, statusPartiallyAvailable)
	n.connectedNodes[act.p] = struct{}{}
	n.currentDownloadBw.update(act.bw)
	act.p.currentUploadBw.update(act.bw)
}

func (n *node) transfer(act action) {
	chunkTransferTime := n.sf.chunkSize / act.bw
	estFinSimTime := n.simTime + chunkTransferTime
	fmt.Printf("%v :Transferring in %.2f seconds...\n", n.id, chunkTransferTime)
	time.Sleep(time.Duration(chunkTransferTime*1000) * time.Millisecond)
	fmt.Printf("%v :Done!\n", n.id)
	n.c <- transferResult{act, estFinSimTime}
}

func (n *node) transferDone(act action) {
	n.sf.setChunk(act.chkId, statusAvailable)
	delete(n.connectedNodes, act.p)
	n.currentDownloadBw.update(-act.bw)
	act.p.currentUploadBw.update(-act.bw)
}
