package main

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

func initializeTestNodes(sv *supervisor) {
	segfileInfo := newSegfileInfo(12*MB, 10, 512*KB)
	var n *node
	for i := 0; i < 10; i++ {
		var ratio float64
		if ratio = 0.4; i == 7 {
			ratio = 1.0
		}
		n = newNode(&segfileInfo, 10*MB, 1-1/math.E, ratio)
		sv.addNode(n)
	}
}

func initializeTestSupervisor() *supervisor {
	sv := supervisor{
		poolLock:    sync.RWMutex{},
		pool:        make(map[*node]struct{}),
		bwRatioLock: sync.RWMutex{},
		bwRatio:     make(map[*node]float64),
		lg:          logger{make(chan []byte)},
	}
	return &sv
}

func getUncompletedNode(sv *supervisor) *node {
	var n *node
	for p := range sv.pool {
		if !p.complete {
			n = p
			break
		}
	}
	return n
}

func TestGetCost(t *testing.T) {
	sv := initializeTestSupervisor()
	initializeTestNodes(sv)
	n := getUncompletedNode(sv)

	a, b, _, p, c := sv.getCost(n, 0, 1)

	if p != nil {
		fmt.Println(a, b, p.id, c)
	} else {
		t.Error("Get cost failed")
	}
}

func TestOptimalAction(t *testing.T) {
	sv := initializeTestSupervisor()
	initializeTestNodes(sv)
	n := getUncompletedNode(sv)

	act := sv.getOptimalAction(n, n.connectedNodes)

	if act.p != nil {
		fmt.Printf("%v <---(c:%v,r:%v)---- %v : %.2f MB/s\n", n.id, act.chkId.cIdx, act.chkId.rIdx, act.p.id, act.bw/MB)
	} else {
		t.Error("Expected action, got none")
	}
}
