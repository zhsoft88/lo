package core

// CDC chunker using Gear hash for content-defined chunk boundaries.
// This ensures that insertions/deletions only affect nearby chunks,
// enabling efficient delta storage for large files.

const gearPolynomial = 0xbf58476d1ce4e5b9

var gearTable [256]uint64

func init() {
	for i := range gearTable {
		h := uint64(i)
		for j := 0; j < 8; j++ {
			if h&1 == 1 {
				h = (h >> 1) ^ gearPolynomial
			} else {
				h >>= 1
			}
		}
		gearTable[i] = h
	}
}

// Chunker splits data into variable-sized chunks using content-defined boundaries.
type Chunker struct {
	minSize int
	maxSize int
	mask    uint64
}

// NewChunker creates a chunker. avgSize determines the target chunk size;
// actual chunk sizes range between minSize and maxSize.
func NewChunker(minSize, avgSize, maxSize int) *Chunker {
	// mask is avgSize-1 rounded to nearest power-of-2-minus-one
	mask := uint64(avgSize - 1)
	return &Chunker{minSize: minSize, maxSize: maxSize, mask: mask}
}

// Chunks splits data into variable-sized pieces and returns their byte slices.
func (c *Chunker) Chunks(data []byte) [][]byte {
	var chunks [][]byte
	start := 0
	for start < len(data) {
		end := c.nextBoundary(data, start)
		chunks = append(chunks, data[start:end])
		start = end
	}
	return chunks
}

// nextBoundary finds the end of the chunk starting at offset start.
func (c *Chunker) nextBoundary(data []byte, start int) int {
	limit := start + c.maxSize
	if limit > len(data) {
		limit = len(data)
	}

	// Don't search for boundary if remaining data fits in minSize
	if limit-start <= c.minSize {
		return limit
	}

	end := start + c.minSize
	hash := uint64(0)

	for end < limit {
		hash = (hash << 1) + gearTable[data[end]]
		if hash&c.mask == 0 {
			return end + 1
		}
		end++
	}

	return limit
}
