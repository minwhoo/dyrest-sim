package main

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

func newNode(pool *[]*node, file *segfile, dlBandwidth float64, ulBandwidth float64, maxDownloadLinks int, maxUploadLinks int) {
	n := node{
		id: 					nodeIdx,
		downloadBandwidth:		dlBandwidth,
		uploadBandwidth:		ulBandwidth,
		maxDownloadLinks:		maxDownloadLinks,
		maxUploadLinks:			maxUploadLinks,
		connectedNodes: 		[]*node{},
		dataChunkAvailability: 	make([]bool,file.numDataChunks),
		parityChunkAvailability:[][]bool{},
		pool:					pool,
		segfile: 				file,
	}
	nodeIdx++
	*pool = append(*pool, &n)
}
