package main

import (
	"time"
	"math/rand"
)

var nodeIdx = 0

type node struct {
	id int
	downloadBandwidth float64
	uploadBandwidth float64
	maxDownloadLinks int
	maxUploadLinks int
	connectedNodes []*node
	dataChunkAvailability []bool
	parityChunkAvailability [][]bool
	pool *[]*node
	segfile *segfile
}

func getRandomAvailability(ratio float64, numChunks int) []bool {
	availability := make([]bool,numChunks)
	rand.Seed(time.Now().UnixNano())

	for _, idx := range rand.Perm(numChunks)[:int(ratio*float64(numChunks))] {
		availability[idx] = true
	}

	return availability
}

func newNode(pool *[]*node, file *segfile, dlBandwidth float64, ulBandwidth float64, maxDownloadLinks int, maxUploadLinks int, availabilityRatio float64) {

	n := node{
		id: 					nodeIdx,
		downloadBandwidth:		dlBandwidth,
		uploadBandwidth:		ulBandwidth,
		maxDownloadLinks:		maxDownloadLinks,
		maxUploadLinks:			maxUploadLinks,
		connectedNodes: 		[]*node{},
		dataChunkAvailability: 	getRandomAvailability(availabilityRatio,file.numDataChunks),
		parityChunkAvailability:[][]bool{},
		pool:					pool,
		segfile: 				file,
	}
	nodeIdx++
	*pool = append(*pool, &n)
}
