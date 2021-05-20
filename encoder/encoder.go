// Author: Leigh Capili
// License: MIT

package encoder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

const defaultBlockSize = 1024

// Encoder represents a single chunkable stream of a file.
// It will automatically open its file on the first Request.
type Encoder struct {
	FileName  string
	BlockSize int64

	cacheKey         string
	file             *os.File
	numBlocks        int64
	highestBlockSize int64
}

func (e *Encoder) PreProcess() (err error) {
	// stat, check file
	info, err := os.Stat(e.FileName)
	if err != nil {
		return
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", e.FileName)
	}

	// populate block info
	e.coerceBlockSize()
	e.numBlocks, e.highestBlockSize = e.getBlockInfo(info.Size())

	// check for cache hit, if not, ensure directory
	err = e.initCacheKey()
	if err != nil {
		return
	}
	cacheDir := e.cacheDir()

	cacheInfo, err := os.Stat(cacheDir)
	if err == nil {
		if !cacheInfo.IsDir() {
			return fmt.Errorf("CacheDir %q is not a directory", cacheDir)
		}
		// cache hit
		fmt.Printf("[encoder] Cache hit for %q\n", e.FileName)
		return
	} else if os.IsNotExist(err) {
		// no cache existing, create one
		if err = os.MkdirAll(cacheDir, 0750); err != nil {
			return
		}
	} else {
		// failed to stat for some reason
		return
	}

	// open
	f, err := os.Open(e.FileName)
	if err != nil {
		return
	}
	defer func() {
		err = f.Close()
	}()

	// the first block we read is the highest block which may be smaller than the rest
	block := make([]byte, e.highestBlockSize)
	// the first/highest block doesn't have a parent hash, it just gets padded with 0's by the encoder
	parentHash := make([]byte, 32)

	// iterate through all block indexes from highest to 0
	for i := e.numBlocks - 1; i >= 0; i-- {
		// seek and read the exact block
		_, err = f.Seek(e.BlockSize*i, os.SEEK_SET)
		if err != nil {
			return
		}
		_, err = f.Read(block)
		// we don't expect an EOF, even on the highest block, because it will successfully read bytes
		if err != nil {
			return
		}

		// use any existing hash with the block to produce the next one
		block = append(block, parentHash...)

		hash := sha256.New()
		_, err = hash.Write(block)
		if err != nil {
			return
		}
		parentHash = hash.Sum(nil)
		err = os.WriteFile(e.hashFile(i), parentHash, 0440)
		if err != nil {
			return
		}

		// reset the block's data and size for the next read, any following reads will be the full block-size
		block = make([]byte, e.BlockSize)
	}

	return
}

// Request will either return the first hash or a block with an appended hash for the next block.
// The client can split these byte sections (being aware of the hash-length) and verify subsequent blocks.
// `FileName` is automatically opened on the first iteration.
// On the final request, the file is closed and an unhashed block is returned, padded with a nil hash of 32 0 bytes.
// The client may attempt to store the final bytes, but it may not make sense
// Subsequent requests will return no bytes and an io.EOF error.
func (e *Encoder) Request(requestNumber int64) ([]byte, error) {
	if requestNumber == 0 {
		// request 0 returns hash 0
		return os.ReadFile(e.hashFile(requestNumber))
	}

	// request 1 returns block 0, request 2 returns block 1
	blockIndex := requestNumber - 1

	if blockIndex >= e.numBlocks {
		return nil, io.EOF
	}

	// ensure file is open
	if e.file == nil {
		fmt.Printf("[encoder] Opening %q\n", e.FileName)
		var err error
		e.file, err = os.Open(e.FileName)
		// there is no accompanying defer for this open file, it will be closed when the client calls e.Close
		if err != nil {
			return nil, err
		}
	}

	_, err := e.file.Seek(e.BlockSize*blockIndex, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	readSize := e.BlockSize
	// last block has potentially smaller block-size
	if blockIndex == e.numBlocks-1 {
		readSize = e.highestBlockSize
	}
	block := make([]byte, readSize)
	_, err = e.file.Read(block)
	if err != nil {
		return block, err
	}

	// if not last block, append parent's hash
	if blockIndex != e.numBlocks-1 {
		hash, err := os.ReadFile(e.hashFile(requestNumber))
		if err != nil {
			return block, err
		}
		block = append(block, hash...)
	} else {
		// pad block with 32 byte long 0-hash
		block = append(block, make([]byte, 32)...)
	}

	return block, err
}

// Close is a helper for the client to end the stream early
func (e *Encoder) Close() error {
	return e.file.Close()
}

func (e *Encoder) initCacheKey() (err error) {
	fpath, err := filepath.Abs(e.FileName)
	if err != nil {
		return
	}
	hash := sha256.New()
	_, err = hash.Write([]byte(fpath))
	if err != nil {
		return
	}
	e.cacheKey = hex.EncodeToString(hash.Sum(nil))
	return
}

func (e *Encoder) cacheDir() string {
	return path.Join("cache", e.cacheKey)
}

func (e *Encoder) hashFile(blockIndex int64) string {
	return path.Join(e.cacheDir(), fmt.Sprintf("%d.sha256", blockIndex))
}

func (e *Encoder) coerceBlockSize() {
	if e.BlockSize <= 0 {
		fmt.Printf("[encoder] Warning: invalid BlockSize %d, defaulting to %d\n", e.BlockSize, defaultBlockSize)
		e.BlockSize = defaultBlockSize
	}
}

func (e *Encoder) getBlockInfo(fileSize int64) (numBlocks, highestBlockSize int64) {
	numBlocks = (fileSize-1)/e.BlockSize + 1
	highestBlockSize = (fileSize-1)%e.BlockSize + 1 // always > 0
	fmt.Printf("[encoder] numBlocks: %d, highestBlockSize: %d\n", numBlocks, highestBlockSize)
	return
}
