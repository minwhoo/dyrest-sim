package main

import (
	"fmt"
	"math/rand"
	"time"
)

var nodeIdx = 0

type req struct {
	n       *node
	chunkId int
	c       chan bool
}

type node struct {
	id                      int
	downloadBandwidth       float64
	uploadBandwidth         float64
	numDownloadLinks        int
	numUploadLinks          int
	maxDownloadLinks        int
	maxUploadLinks          int
	connectedNodes          []*node
	dataChunkAvailability   []int
	parityChunkAvailability [][]int
	pool                    *[]*node
	segfile                 *segfile
	down                    chan int
	up                      chan int
	c                       chan req
}

func getRandomAvailability(ratio float64, numChunks int) []int {
	availability := make([]int, numChunks)
	rand.Seed(time.Now().UnixNano())

	for _, idx := range rand.Perm(numChunks)[:int(ratio*float64(numChunks))] {
		availability[idx] = 2
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
		connectedNodes:          []*node{},
		dataChunkAvailability:   getRandomAvailability(availabilityRatio, file.numDataChunks),
		parityChunkAvailability: [][]int{},
		pool:    pool,
		segfile: file,
		down:    make(chan int),
		up:      make(chan int),
		c:       make(chan req),
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

func (n *node) transfer(p *node, chunkId int) {
	//chunkTransferTime := n.segfile.chunkSize / math.Min(n.downloadBandwidth, n.uploadBandwidth)
	transferTime := time.Duration(200+rand.Intn(800)) * time.Millisecond
	fmt.Printf("Tranferring chunk %v from node %v to node %v in %v milliseconds...\n", chunkId, p.id, n.id, transferTime)
	time.Sleep(transferTime)
	fmt.Println("Done!")
	n.down <- chunkId
	p.up <- chunkId
}

func (n *node) downloadLoop(c chan bool) {
	reqc := make(chan bool)
	for {
		p, chunkId := n.evaluateBestChunkPair()
		if chunkId == -1 {
			n.dataChunkAvailability[<-n.down] = 2
			if n.checkDownloadComplete() {
				c <- true
			}
		} else {
			if n.numDownloadLinks < n.maxDownloadLinks {
				n.request(p, chunkId, reqc)
				if <-reqc {
					n.dataChunkAvailability[chunkId] = 1
					go n.transfer(p, chunkId)
					n.numDownloadLinks++
				}
			} else {
				n.dataChunkAvailability[<-n.down] = 2
				fmt.Println("Transferring finally finished!!")
				n.numDownloadLinks--
			}
			select {
			case val := <-n.down:
				n.dataChunkAvailability[val] = 2
				fmt.Println("Transferring finished!!")
				n.numDownloadLinks--
			default:
			}
		}
	}
}

func (n *node) evaluateBestChunkPair() (p *node, chunkId int) {
	p = (*n.pool)[0]
	chunkId = -1
	for i := 0; i < n.segfile.numDataChunks; i++ {
		if n.dataChunkAvailability[i] == 0 && p.dataChunkAvailability[i] == 2 {
			chunkId = i
			return
		}
	}
	return
}

func (n *node) listen() {
	for {
		select {
		case msg := <-n.c:
			fmt.Println("received!", msg.chunkId)
			n.respond(msg)
		}
	}
}

func (n *node) request(p *node, chunkId int, c chan bool) {
	fmt.Println("Requesting chunk", chunkId, "...")
	p.c <- req{n, chunkId, c}
}

func (n *node) respond(msg req) {
	msg.c <- true
}
