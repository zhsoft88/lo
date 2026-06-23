package repo

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zhsoft88/lo/internal/core"
)

// StoreObject serializes, compresses, and writes an object to the content-addressable store.
// Returns the SHA256 hash of the uncompressed content (before the header/compression).
func (r *Repository) StoreObject(objType core.ObjectType, content []byte) (core.Hash, error) {
	data, err := core.SerializeObject(objType, content)
	if err != nil {
		return core.Hash{}, fmt.Errorf("serialize object: %w", err)
	}

	h := core.HashFromBytes(data)
	objPath := r.objectPath(h)

	if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
		return core.Hash{}, fmt.Errorf("create object dir: %w", err)
	}

	if err := ioutil.WriteFile(objPath, data, 0644); err != nil {
		return core.Hash{}, fmt.Errorf("write object: %w", err)
	}

	return h, nil
}

// LoadObject reads, decompresses, and deserializes an object from the store.
func (r *Repository) LoadObject(hash core.Hash) (core.ObjectType, []byte, error) {
	objPath := r.objectPath(hash)
	data, err := ioutil.ReadFile(objPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, fmt.Errorf("object not found: %s", hash)
		}
		return 0, nil, fmt.Errorf("read object: %w", err)
	}

	objType, content, err := core.DeserializeObject(data)
	if err != nil {
		return 0, nil, fmt.Errorf("deserialize object: %w", err)
	}

	return objType, content, nil
}

// HasObject checks if an object exists in the store.
func (r *Repository) HasObject(hash core.Hash) bool {
	_, err := os.Stat(r.objectPath(hash))
	return err == nil
}

// objectPath returns the filesystem path for an object hash.
// Uses the git-style XX/YYYYYY layout: first two hex chars as directory.
func (r *Repository) objectPath(hash core.Hash) string {
	s := hash.String()
	return filepath.Join(r.ObjectsDir(), s[:2], s[2:])
}

// ObjectType reads the type of a stored object by peeking at the first
// byte of the uncompressed header without fully decompressing the content.
func (r *Repository) ObjectType(hash core.Hash) (core.ObjectType, error) {
	data, err := ioutil.ReadFile(r.objectPath(hash))
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("empty object file")
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("decompress header: %w", err)
	}
	defer gr.Close()

	typeByte := make([]byte, 1)
	if _, err := io.ReadFull(gr, typeByte); err != nil {
		return 0, fmt.Errorf("read type byte: %w", err)
	}
	return core.ObjectType(typeByte[0]), nil
}
