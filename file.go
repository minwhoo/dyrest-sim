package main

const (
	_          = iota
	KB float64 = 1 << (10 * iota)
	MB
	GB
	TB
)

type chunk struct {
	idx int
	r   int
}

type segment struct {
	idx          int
	dataChunks   []chunk
	parityChunks [][]chunk
}

type segfile struct {
	segments      []segment
	numSegments   int
	numDataChunks int
	chunkSize     float64
}

// newSegfile creates segfile based on file size, segment length, and chunk size
// the unit for file size and chunk size should both be bytes, and chunk size should be evenly dividable by file size
// the segment length denotes the number of data chunks in a segment
func newSegfile(fileSize float64, segmentLength int, chunkSize float64) segfile {
	numChunks := int(fileSize / chunkSize)
	numSegments := numChunks / segmentLength
	rem := numChunks % segmentLength

	if rem != 0 {
		numSegments += 1
	}

	segments := make([]segment, numSegments)

	for i := 0; i < numSegments; i++ {
		segLen := segmentLength
		if i == numSegments-1 && rem != 0 {
			segLen = rem
		}
		chunks := make([]chunk, segLen)
		for j := 0; j < segLen; j++ {
			chunks[j] = chunk{idx: i*segmentLength + j, r: 0}
		}

		segments[i] = segment{i, chunks, [][]chunk{}}
	}

	return segfile{segments, numSegments, numChunks, float64(chunkSize)}
}
