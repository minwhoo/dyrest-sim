package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var nodeIdx = 0

type req struct {
	n   *node
	chk chunk
	c   chan bool
}

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
	currentDownloadBw       float64
	currentUploadBw         float64
	connectedNodes          map[*node]struct{}
	dataChunkAvailability   []availabilityStatus
	parityChunkAvailability [][]availabilityStatus
	downc                   chan action
	upc                     chan int
	reqc                    chan req
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
		currentDownloadBw:       0,
		currentUploadBw:         0,
		connectedNodes:          make(map[*node]struct{}),
		dataChunkAvailability:   getRandomAvailability(availabilityRatio, sv.file.numDataChunks),
		parityChunkAvailability: [][]availabilityStatus{},
		downc:    make(chan action),
		upc:      make(chan int),
		reqc:     make(chan req),
		complete: false,
	}
	if availabilityRatio == 1 {
		n.complete = true
	}
	nodeIdx++
	return &n
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
	go n.listen()
	if !n.complete {
		wg.Add(1)
		go n.downloadLoop(wg)
	}
}

func (n *node) transfer(act action) {
	//chunkTransferTime := time.Duration(n.supervisor.file.chunkSize/act.bw*1000) * time.Millisecond
	transferTime := time.Duration(200+rand.Intn(800)) * time.Millisecond
	//fmt.Printf(":Tranferring chunk %v from node %v to node %v in %v milliseconds...\n", act.chk.idx, act.p.id, n.id, chunkTransferTime)
	time.Sleep(transferTime)
	//fmt.Println("Done!")
	n.downc <- act
	act.p.upc <- 1
}

func (n *node) downloadLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	resc := make(chan bool)
	var act action
	for {
		if na, pa, _ := n.countAvailability(); na == 0 {
			if pa == 0 {
				fmt.Println(n.id, ": Download complete!")
				n.complete = true
				break
			}

			goto block
		}

		act = n.supervisor.getOptimalAction(n, n.connectedNodes, n.maxBw-n.currentDownloadBw)

		if act.p == nil {
			goto block
		}

		fmt.Println(n.id, ": node target", act.p.id, "selected")

		n.request(act, resc)
		if <-resc {
			n.setAvailability(act.chk, statusPartiallyAvailable)
			n.connectedNodes[act.p] = struct{}{}
			n.currentDownloadBw += act.bw
			fmt.Println(n.id, ": tb current bw(MB): ", n.currentDownloadBw/MB)
			go n.transfer(act)
		}

		continue

	block:
		completedAct := <-n.downc
		n.setAvailability(completedAct.chk, statusAvailable)
		delete(n.connectedNodes, completedAct.p)
		n.currentDownloadBw -= completedAct.bw
		fmt.Println(n.id, ": td current bw(MB): ", n.currentDownloadBw/MB)
	}
}

func (n *node) setAvailability(chk chunk, status availabilityStatus) {
	n.dataChunkAvailability[chk.idx] = status
	n.supervisor.updateAvailability(n, chk, status)
}

func (n *node) listen() {
	for {
		select {
		case msg := <-n.reqc:
			fmt.Println("received!", msg.chk.idx)
			n.respond(msg)
		}
	}
}

func (n *node) request(act action, c chan bool) {
	fmt.Println(n.id, ":Requesting chunk", act.chk.idx, "...")
	act.p.reqc <- req{n, act.chk, c}
}

func (n *node) respond(msg req) {
	msg.c <- true
}
