package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const HashSize = 32

type Hash [HashSize]byte

func HashFromBytes(data []byte) Hash {
	return sha256.Sum256(data)
}

func HashFromHex(s string) (Hash, error) {
	if len(s) != HashSize*2 {
		return Hash{}, fmt.Errorf("invalid hash length: %d", len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return Hash{}, err
	}
	var h Hash
	copy(h[:], b)
	return h, nil
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) Short() string {
	return hex.EncodeToString(h[:8])
}

func (h Hash) IsZero() bool {
	return h == Hash{}
}

func (h Hash) MarshalJSON() ([]byte, error) {
	return []byte(`"` + h.String() + `"`), nil
}

func (h *Hash) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("invalid hash JSON: %s", b)
	}
	parsed, err := HashFromHex(string(b[1 : len(b)-1]))
	if err != nil {
		return err
	}
	*h = parsed
	return nil
}

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}
