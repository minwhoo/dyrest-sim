package main

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type supervisor struct {
	poolLock    sync.RWMutex
	pool        map[*node]struct{}
	bwRatioLock sync.RWMutex
	bwRatio     map[*node]float64
	lg          logger
}

type action struct {
	p     *node
	chkId chunkId
	bw    float64
}

func (sv *supervisor) addNode(n *node) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	sv.poolLock.Lock()
	sv.pool[n] = struct{}{}
	sv.poolLock.Unlock()

	sv.bwRatioLock.Lock()
	//sv.bwRatio[n] = math.Min(math.Max(0.1, random.NormFloat64()*0.3+0.6), 0.9)
	sv.bwRatio[n] = 0.1 + 0.9*random.Float64()
	sv.bwRatioLock.Unlock()

	//sv.lg.logNodeAdded(n)
}

func (sv *supervisor) removeNode(n *node) {
	sv.poolLock.Lock()
	delete(sv.pool, n)
	sv.poolLock.Unlock()

	sv.bwRatioLock.Lock()
	delete(sv.bwRatio, n)
	sv.bwRatioLock.Unlock()
}

func (sv *supervisor) getFastOptimalAction(n *node, connectedNodes map[*node]struct{}) action {
	sv.poolLock.RLock()
	defer sv.poolLock.RUnlock()

	for p := range sv.pool {
		if _, ok := connectedNodes[p]; !ok && n != p {
			for sIdx := 0; sIdx < n.sf.numSegments; sIdx++ {
				if n.sf.segments[sIdx].complete != true {
					for chkIdx, val := range n.sf.segments[sIdx].chunks[0] {
						if val == statusNotAvailable {
							bw, err := sv.getBandwidth(n, p)
							if !err {
								return action{p, chunkId{sIdx, 0, chkIdx}, bw}
							}
						}
					}

					//fmt.Println(n.sf.segments[sIdx])
				}
			}
		}
	}
	return action{nil, chunkId{0, 0, 0}, 0}
}

func (sv *supervisor) getBandwidth(n *node, p *node) (bw float64, err bool) {
	sv.bwRatioLock.RLock()
	defer sv.bwRatioLock.RUnlock()
	maxDownloadThroughput := n.getMaxDownloadBw() - n.currentDownloadBw.get()
	maxUploadThroughput := p.getMaxUploadBw() - p.currentUploadBw.get()
	bw = math.Min(maxDownloadThroughput, maxUploadThroughput) * sv.bwRatio[p] // b n choose two
	err = false
	/*
		if bw < n.getMaxDownloadBw()/10 {
			bw = 0
			err = true
		}
	*/
	return
}

func (sv *supervisor) getOptimalAction(n *node, connectedNodes map[*node]struct{}) action {
	sv.poolLock.RLock()
	defer sv.poolLock.RUnlock()

	for sIdx := 0; sIdx < n.sf.numSegments; sIdx++ {
		if n.sf.segments[sIdx].complete != true {
			var minP *node
			var minCIdx int
			var minRIdx int
			var bw float64
			minCost := math.Inf(1)

			for r := 1; r <= 3; r++ {
				_, b, _, p, c := sv.getCost(n, sIdx, r)
				if b < minCost {
					minP = p
					minCIdx = c % n.sf.getSegmentSize(sIdx)
					minRIdx = c / n.sf.getSegmentSize(sIdx)
					minCost = b
				}
			}

			if minP != nil {
				bw, _ = sv.getBandwidth(n, minP)
			} else {
				bw = 0
			}
			return action{minP, chunkId{sIdx, minRIdx, minCIdx}, bw}
		}
	}

	return action{nil, chunkId{0, 0, 0}, 0}
}

func (sv *supervisor) getCost(n *node, sIdx int, rIdx int) (float64, float64, bool, *node, int) {
	var data [][]float64
	var ref []*node
	var numAllChunks int
	pIdx := 0

	nChunks := append(n.sf.getChunks(sIdx, 0), n.sf.getChunks(sIdx, rIdx)...)
	numAllChunks = len(nChunks)

	for p := range sv.pool {
		if _, ok := n.connectedNodes[p]; n == p || ok { // unconnected + not me
			continue
		}

		pChunks := append(p.sf.getChunks(sIdx, 0), p.sf.getChunks(sIdx, rIdx)...)
		data = append(data, make([]float64, numAllChunks))
		ref = append(ref, p)

		// *** BEWARE OF CALCULATING EXACT BANDWIDTH! ***
		bandwidth, _ := sv.getBandwidth(n, p)

		for cIdx := range pChunks {

			var atrc float64 = 1

			if nChunks[cIdx] != statusNotAvailable {
				atrc = 0
			}

			cost := math.Inf(1)
			if pChunks[cIdx] == statusAvailable {
				if cIdx < n.sf.getSegmentSize(sIdx) {
					cost = 1
				} else {
					cost = 1.05
				}
			} else if n.sf.segments[sIdx].complete {
				cost = 1.11
			}

			data[pIdx][cIdx] = (1 / cost) * atrc * bandwidth
		}

		pIdx++
	}
	numPeers := len(data)

	// 각 row에서 가장 작은 걸 택하고, 모두 더한다.

	costVec := make([]float64, numAllChunks)

	var totalSeqCost float64
	var totalPrlCost float64

	var minNode *node
	var minACIdx int

	for acIdx := 0; acIdx < numAllChunks; acIdx++ {
		minCost := math.Inf(1)
		for pIdx := 0; pIdx < numPeers; pIdx++ {
			// Do not check null valued chunk
			if data[pIdx][acIdx] == 0 {
				continue
			}

			cost := 1 / data[pIdx][acIdx]

			if cost < minCost {
				minCost = cost
				minNode = ref[pIdx]
				minACIdx = acIdx
			}
		}

		costVec[acIdx] = minCost
	}

	sort.Float64s(costVec)

	// Update cost values
	broken := false

	for nrow := 0; nrow < len(costVec)-rIdx/2; nrow++ {
		cost := costVec[nrow]

		if math.IsInf(cost, 1) {
			broken = true
			continue
		}

		totalSeqCost += cost
		if totalPrlCost < cost {
			totalPrlCost = cost
		}
	}

	return totalSeqCost, totalPrlCost, broken, minNode, minACIdx
}
