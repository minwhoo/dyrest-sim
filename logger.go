package main

import (
	"encoding/json"
)

type logger struct {
	c chan []byte
}

type Message struct {
	Code int
	Data interface{}
}

type NodeData struct {
	Id           int
	Availability []segment
}

type NodeAvailabilityData struct {
	Id     int
	Chunk  ChunkData
	Status availabilityStatus
}
type ChunkData struct {
	Idx int
	R   int
}

const (
	MessageNodeAdded int = iota
	MessageNodeAvailibilityUpdated
)

func (lg logger) logNodeAdded(n *node) {
	data := NodeData{n.id, n.sf.segments}
	msg, err := json.Marshal(Message{MessageNodeAdded, data})
	if err == nil {
		lg.c <- msg
	}
}

func (lg logger) logNodeAvailabilityUpdated(id int, chk chunkId, status availabilityStatus) {
	data := NodeAvailabilityData{
		id,
		ChunkData{
			chk.cIdx,
			chk.rIdx,
		},
		status,
	}
	msg, err := json.Marshal(Message{MessageNodeAvailibilityUpdated, data})
	if err == nil {
		lg.c <- msg
	}
}
