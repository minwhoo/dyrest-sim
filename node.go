package main

import (
	"fmt"
	"math/rand"
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
	downloadBandwidth       float64
	uploadBandwidth         float64
	numDownloadLinks        int
	numUploadLinks          int
	maxDownloadLinks        int
	maxUploadLinks          int
	connectedNodes          map[*node]struct{}
	dataChunkAvailability   []availabilityStatus
	parityChunkAvailability [][]availabilityStatus
	pool                    *[]*node
	segfile                 *segfile
	downc                   chan chunk
	upc                     chan int
	reqc                    chan req
}

func getRandomAvailability(ratio float64, numChunks int) []availabilityStatus {
	availability := make([]availabilityStatus, numChunks)
	rand.Seed(time.Now().UnixNano())

	for _, idx := range rand.Perm(numChunks)[:int(ratio*float64(numChunks))] {
		availability[idx] = statusAvailable
	}

	return availability
}

func newNode(pool *[]*node, file *segfile, dlBandwidth float64, ulBandwidth float64, maxDownloadLinks int, maxUploadLinks int, availabilityRatio float64) {

	n := node{
		id:                      nodeIdx,
		downloadBandwidth:       dlBandwidth,
		uploadBandwidth:         ulBandwidth,
		numDownloadLinks:        0,
		numUploadLinks:          0,
		maxDownloadLinks:        maxDownloadLinks,
		maxUploadLinks:          maxUploadLinks,
		connectedNodes:          make(map[*node]struct{}),
		dataChunkAvailability:   getRandomAvailability(availabilityRatio, file.numDataChunks),
		parityChunkAvailability: [][]availabilityStatus{},
		pool:    pool,
		segfile: file,
		downc:   make(chan chunk),
		upc:     make(chan int),
		reqc:    make(chan req),
	}
	nodeIdx++
	*pool = append(*pool, &n)
}

func (n *node) checkDownloadComplete() bool {
	for idx, val := range n.dataChunkAvailability {
		if val < 2 {
			fmt.Println(idx, "not finished")
			return false
		}
	}
	return true
}

func (n *node) startSeed() {
	go n.listen()
}

func (n *node) startDownload(c chan bool) {
	go n.downloadLoop(c)
}

func (n *node) transfer(p *node, chk chunk) {
	//chunkTransferTime := time.Duration(n.segfile.chunkSize / math.Min(n.downloadBandwidth, p.uploadBandwidth))
	transferTime := time.Duration(200+rand.Intn(800)) * time.Millisecond
	fmt.Printf("Tranferring chunk %v from node %v to node %v in %v milliseconds...\n", chk.idx, p.id, n.id, transferTime)
	time.Sleep(transferTime)
	fmt.Println("Done!")
	n.downc <- chk
	p.upc <- 1
}

func (n *node) downloadLoop(c chan bool) {
	resc := make(chan bool)
	for {
		p, chk := n.evaluateBestChunkPair()
		if p == nil {
			if n.checkDownloadComplete() {
				c <- true
			} else {
				n.setAvailability(<-n.downc, statusAvailable)
			}
		} else {
			if n.numDownloadLinks < n.maxDownloadLinks {
				n.request(p, chk, resc)
				if <-resc {
					n.setAvailability(chk, statusPartiallyAvailable)
					go n.transfer(p, chk)
					n.numDownloadLinks++
				}
			} else {
				// blocking wait
				n.setAvailability(<-n.downc, statusAvailable)
				n.numDownloadLinks--
			}

			// non-blocking check
			select {
			case downchk := <-n.downc:
				n.setAvailability(downchk, statusAvailable)
				n.numDownloadLinks--
			default:
			}
		}
	}
}

func (n *node) setAvailability(chk chunk, status availabilityStatus) {
	n.dataChunkAvailability[chk.idx] = status
}

func (n *node) evaluateBestChunkPair() (p *node, chk chunk) {
	for i := 0; i < n.segfile.numDataChunks; i++ {
		for idx, iterp := range *n.pool {
			if idx != n.id && n.dataChunkAvailability[i] == statusNotAvailable && iterp.dataChunkAvailability[i] == statusAvailable {
				return iterp, chunk{i, 0}
			}
		}
	}
	return nil, chunk{0, 0}
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

func (n *node) request(p *node, chk chunk, c chan bool) {
	fmt.Println(n.id, ":Requesting chunk", chk.idx, "...")
	p.reqc <- req{n, chk, c}
}

func (n *node) respond(msg req) {
	msg.c <- true
}
