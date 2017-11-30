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
	downloadBandwidth       float64
	uploadBandwidth         float64
	numDownloadLinks        int
	numUploadLinks          int
	maxDownloadLinks        int
	maxUploadLinks          int
	connectedNodes          map[*node]struct{}
	dataChunkAvailability   []availabilityStatus
	parityChunkAvailability [][]availabilityStatus
	downc                   chan chunk
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

func newNode(sv *supervisor, dlBandwidth float64, ulBandwidth float64, maxDownloadLinks int, maxUploadLinks int, availabilityRatio float64) *node {
	n := node{
		id:                      nodeIdx,
		supervisor:              sv,
		downloadBandwidth:       dlBandwidth,
		uploadBandwidth:         ulBandwidth,
		numDownloadLinks:        0,
		numUploadLinks:          0,
		maxDownloadLinks:        maxDownloadLinks,
		maxUploadLinks:          maxUploadLinks,
		connectedNodes:          make(map[*node]struct{}),
		dataChunkAvailability:   getRandomAvailability(availabilityRatio, sv.file.numDataChunks),
		parityChunkAvailability: [][]availabilityStatus{},
		downc:    make(chan chunk),
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

func (n *node) checkDownloadComplete() bool {
	for _, val := range n.dataChunkAvailability {
		if val < 2 {
			//fmt.Println(idx, "not finished")
			return false
		}
	}
	return true
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
	n.numDownloadLinks++
	//chunkTransferTime := time.Duration(n.segfile.chunkSize / math.Min(n.downloadBandwidth, p.uploadBandwidth))
	transferTime := time.Duration(200+rand.Intn(800)) * time.Millisecond
	//fmt.Printf("Tranferring chunk %v from node %v to node %v in %v milliseconds...\n", act.chk.idx, act.p.id, n.id, transferTime)
	time.Sleep(transferTime)
	//fmt.Println("Done!")
	n.numDownloadLinks--
	n.downc <- act.chk
	act.p.upc <- 1
}

func (n *node) downloadLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	resc := make(chan bool)
	for {
		act := n.supervisor.getOptimalAction(n, n.connectedNodes)
		if act.p == nil {
			if n.checkDownloadComplete() {
				fmt.Println(n.id, ": Download complete!")
				n.complete = true
				break
			} else {
				n.setAvailability(<-n.downc, statusAvailable)
			}
		} else {
			fmt.Println(n.id, ": node target", act.p.id, "selected")
			if n.numDownloadLinks < n.maxDownloadLinks {
				n.request(act, resc)
				if <-resc {
					n.setAvailability(act.chk, statusPartiallyAvailable)
					go n.transfer(act)
				}
			} else {
				// blocking wait
				n.setAvailability(<-n.downc, statusAvailable)
			}

			// non-blocking check
			select {
			case chk := <-n.downc:
				n.setAvailability(chk, statusAvailable)
			default:
			}
		}
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
