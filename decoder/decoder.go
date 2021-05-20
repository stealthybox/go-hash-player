package decoder

import (
	"crypto/sha256"
	"fmt"
)

// Decode takes in Encoded bytes and outputs Decoded bytes.
// It verifies that decoded blocks are cryptographically related using the input hash.
// On valid blocks, it returns the block along with the hash of the next related block.
func Decode(hash, hashedBlock []byte) (block, nextHash []byte, err error) {
	hashOffset := len(hashedBlock) - 32
	if hashOffset <= 0 {
		return nil, nil, fmt.Errorf("Hashed block too short, expected length > 32, got: %v", len(hashedBlock))
	}

	// verify
	h := sha256.New()
	h.Write(hashedBlock)
	clientHash := h.Sum(nil)

	for i := 0; i < 32; i++ {
		if clientHash[i] != hash[i] {
			return nil, nil, fmt.Errorf("Hashed block failed verification, expected: %v, got: %v", hash, clientHash)
		}
	}

	// the bytes are valid, so set the nextHash
	block = hashedBlock[:hashOffset]
	nextHash = hashedBlock[hashOffset:]

	return
}
