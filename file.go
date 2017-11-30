package main

const (
	_	   = iota
	KB int = 1 << (10 * iota)
	MB
	GB
	TB
)

type chunk struct {
	id int
}

type segment struct {
	id           int
	dataChunks   []chunk
	parityChunks [][]chunk
}

type segfile struct {
	segments    []segment
	numSegments int
	numDataChunks int
	chunkSize int
}

// newSegfile creates segfile based on file size, segment length, and chunk size
// the unit for file size and chunk size should both be bytes, and chunk size should be evenly dividable by file size
// the segment length denotes the number of data chunks in a segment
func newSegfile(fileSize int, segmentLength int, chunkSize int) segfile {
	numChunks := fileSize / chunkSize
	numSegments := numChunks / segmentLength
	r := numChunks % segmentLength

	if r != 0 {
		numSegments += 1
	}

	segments := make([]segment, numSegments)

	for i := 0; i < numSegments; i++ {
		segLen := segmentLength
		if i == numSegments-1 && r != 0 {
			segLen = r
		}
		chunks := make([]chunk, segLen)
		for j := 0; j < segLen; j++ {
			chunks[j] = chunk{id: i*segmentLength + j}
		}

		segments[i] = segment{i, chunks, [][]chunk{}}
	}

	return segfile{segments, numSegments, numChunks, chunkSize}
}
