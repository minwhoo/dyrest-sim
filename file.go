package main

import (
	"fmt"
	"math"
	"sync"
)

const (
	_          = iota
	KB float64 = 1 << (10 * iota)
	MB
	GB
	TB
)

const (
	originalChunk int = iota
	generatedChunk
	generatableChunk
)

type chunkId struct {
	sIdx int
	rIdx int
	cIdx int
}

type segment struct {
	idx             int
	chunks          [][]availabilityStatus // redundancy level, chunk index
	remaining       []int
	transferring    int
	plannedComplete bool
	complete        bool
}

type segfile struct {
	*segfileInfo
	segments          []segment
	numTransferChunks int
	lock              sync.RWMutex
}

type segfileInfo struct {
	numSegments   int
	numDataChunks int
	segmentSize   int
	chunkSize     float64
	fileSize      float64
}

func (sfinfo *segfileInfo) getSegmentSize(sIdx int) int {
	r := sfinfo.numDataChunks % sfinfo.segmentSize
	if r != 0 && sIdx == sfinfo.numSegments-1 {
		return r
	}
	return sfinfo.segmentSize
}

func newSegfileInfo(fileSize float64, segmentSize int, chunkSize float64) segfileInfo {
	numChunks := int(fileSize / chunkSize)
	numSegments := int(math.Ceil(float64(numChunks) / float64(segmentSize)))
	return segfileInfo{numSegments, numChunks, segmentSize, float64(chunkSize), fileSize}
}

func newSegfile(sfinfo *segfileInfo) segfile {
	sf := segfile{
		sfinfo,
		make([]segment, sfinfo.numSegments),
		0,
		sync.RWMutex{},
	}
	for i := 0; i < sfinfo.numSegments; i++ {
		sf.segments[i] = segment{i, make([][]availabilityStatus, 1), []int{sf.getSegmentSize(i)}, 0, false, false}
		sf.segments[i].chunks[0] = make([]availabilityStatus, sfinfo.getSegmentSize(i))
	}
	return sf
}

func (sf *segfile) getChunks(sIdx int, rIdx int) []availabilityStatus {
	sf.lock.RLock()
	defer sf.lock.RUnlock()
	if len(sf.segments[sIdx].chunks)-1 < rIdx {
		return make([]availabilityStatus, 2*rIdx)
	}
	return sf.segments[sIdx].chunks[rIdx]
}

func (sf *segfile) getTransferChunks() int {
	sf.lock.Lock()
	defer sf.lock.Unlock()

	return sf.numTransferChunks
}

func (sf *segfile) showSegments() {
	sf.lock.RLock()
	defer sf.lock.RUnlock()

	for _, seg := range sf.segments {
		fmt.Printf("%v ", seg.chunks)
	}
	fmt.Printf("\n")
}

func (sf *segfile) transferInProgress() bool {
	sf.lock.RLock()
	sf.lock.RUnlock()

	for _, seg := range sf.segments {
		if seg.transferring > 0 {
			return true
		}
	}

	return false
}

// setChunk is only called by owner node
func (sf *segfile) setChunk(chkId chunkId, newStatus availabilityStatus) {
	sf.lock.Lock()
	defer sf.lock.Unlock()
	prevStatus := sf.segments[chkId.sIdx].chunks[chkId.rIdx][chkId.cIdx]
	// check r-level slice exists, if not allocate slice
	for r := len(sf.segments[chkId.sIdx].chunks); r-1 < chkId.rIdx; r++ {
		sf.segments[chkId.sIdx].chunks = append(sf.segments[chkId.sIdx].chunks, make([]availabilityStatus, 2*r))
		sf.segments[chkId.sIdx].remaining = append(sf.segments[chkId.sIdx].remaining, 2*r)
	}

	// check transfer status
	if newStatus == statusPartiallyAvailable {
		sf.segments[chkId.sIdx].transferring++
	} else if prevStatus == statusPartiallyAvailable {
		sf.segments[chkId.sIdx].transferring--
	}

	// set chunk
	sf.segments[chkId.sIdx].chunks[chkId.rIdx][chkId.cIdx] = newStatus

	// check remaining chunks
	if newStatus == statusPartiallyAvailable {
		if sf.checkRemaining(chkId.sIdx) == sf.segments[chkId.sIdx].transferring {
			sf.segments[chkId.sIdx].plannedComplete = true
		}
	} else if newStatus == statusAvailable {
		sf.segments[chkId.sIdx].remaining[chkId.rIdx]--
		if sf.checkRemaining(chkId.sIdx) == 0 {
			for i := 0; i < sf.getSegmentSize(chkId.sIdx); i++ {
				sf.segments[chkId.sIdx].chunks[0][i] = statusAvailable
			}
			sf.segments[chkId.sIdx].complete = true
		}
	}
}

func (sf *segfile) checkRemaining(sIdx int) int {
	// must call by other function that locks the segfile
	remaining := sf.segments[sIdx].remaining[0]
	minRemaining := remaining
	for r := 1; r < len(sf.segments[sIdx].chunks); r++ {
		rRemaining := remaining + sf.segments[sIdx].remaining[r] - r
		if rRemaining < minRemaining {
			minRemaining = rRemaining
		}
	}
	return minRemaining
}

func (sf *segfile) plannedComplete() bool {
	sf.lock.RLock()
	defer sf.lock.RUnlock()
	for i := 0; i < sf.numSegments; i++ {
		if !sf.segments[i].plannedComplete {
			return false
		}
	}
	return true
}
