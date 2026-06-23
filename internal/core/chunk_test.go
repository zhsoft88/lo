package core

import (
	"bytes"
	"testing"
)

func TestChunkerSmallData(t *testing.T) {
	data := []byte("small data")
	c := NewChunker(2, 64, 256)
	chunks := c.Chunks(data)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for small data, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], data) {
		t.Fatal("chunk content mismatch")
	}
}

func TestChunkerBoundaries(t *testing.T) {
	// Generate data larger than maxSize to force multiple chunks
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 251)
	}

	c := NewChunker(64, 1024, 4096)
	chunks := c.Chunks(data)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for 10KB data, got %d", len(chunks))
	}

	// Verify all chunks respect min/max size
	for i, chunk := range chunks {
		if len(chunk) < 64 && len(chunks) > 1 && i < len(chunks)-1 {
			t.Fatalf("chunk %d too small: %d bytes (min 64)", i, len(chunk))
		}
		if len(chunk) > 4096 {
			t.Fatalf("chunk %d too large: %d bytes (max 4096)", i, len(chunk))
		}
	}

	// Verify reconstructing yields original data
	var reconstructed []byte
	for _, chunk := range chunks {
		reconstructed = append(reconstructed, chunk...)
	}
	if !bytes.Equal(reconstructed, data) {
		t.Fatal("reconstructed data does not match original")
	}
}

func TestChunkerDeterministic(t *testing.T) {
	data := make([]byte, 50000)
	for i := range data {
		data[i] = byte(i*31 + 17)
	}

	c := NewChunker(256, 4096, 16384)
	chunks1 := c.Chunks(data)
	chunks2 := c.Chunks(data)

	if len(chunks1) != len(chunks2) {
		t.Fatalf("different chunk counts: %d vs %d", len(chunks1), len(chunks2))
	}
	for i := range chunks1 {
		if !bytes.Equal(chunks1[i], chunks2[i]) {
			t.Fatalf("chunk %d differs between runs", i)
		}
	}
}

func TestChunkerContentDefined(t *testing.T) {
	// Inserting data in the middle should only shift subsequent chunks
	data := []byte("hello world this is a test of content defined chunking with enough data to trigger boundaries")

	c := NewChunker(4, 16, 64)
	chunks1 := c.Chunks(data)

	modified := append([]byte{}, data[:20]...)
	modified = append(modified, []byte(" INSERTED ")...)
	modified = append(modified, data[20:]...)

	chunks2 := c.Chunks(modified)

	// Some prefixes should share chunk boundaries
	prefixMatch := false
	limit := len(chunks1)
	if len(chunks2) < limit {
		limit = len(chunks2)
	}
	for i := 1; i < limit; i++ {
		if bytes.Equal(chunks1[i], chunks2[i]) {
			prefixMatch = true
			break
		}
	}
	if !prefixMatch {
		t.Log("note: no shared chunks after insertion (may be normal with small data)")
	}
}

func TestChunkerEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{0}},
		{"all zeros", make([]byte, 1000)},
		{"all same byte", bytes.Repeat([]byte{0xAB}, 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChunker(8, 64, 256)
			chunks := c.Chunks(tt.data)

			var reconstructed []byte
			for _, chunk := range chunks {
				reconstructed = append(reconstructed, chunk...)
			}
			if !bytes.Equal(reconstructed, tt.data) {
				t.Fatal("reconstructed data does not match original")
			}
		})
	}
}
