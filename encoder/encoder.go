// Author: Leigh Capili
// License: MIT

package encoder

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

const defaultBlockSize = 1024

// Encoder represents a single chunkable stream of a file.
// It will automatically open its file on the first request.
// It will automatically close its file on the last request.
type Encoder struct {
	FileName  string
	BlockSize int

	finished bool
	file     *os.File

	nextBlock []byte
}

// Request will either return the first hash or a block with an appended hash for the next block.
// The client can split these byte sections (being aware of the hash-length) and verify subsequent blocks.
// `FileName` is automatically opened on the first iteration.
// On the final request, the file is closed and an unhashed block + an io.EOF error is returned.
// Subsequent requests will return no bytes and an io.EOF error.
func (e *Encoder) Request() ([]byte, error) {

	// SOLUTION NOTES:
	// 	This implementation incorrectly solves for the request workflow.
	//  It just-in-time hashes blocks as requests are made.
	//  This function should actually be re-tooled as a pre-processing loop.
	//
	// 	To Process the file we should seek to the highest block offest possible on the first iteration:
	//		( filesize mod blocksize * blocksize )
	// 	Then read BlockSize and seek backwards instead of forwards through the file.
	//  Hashes should include the content of the later blocks including any other appended hashes.
	//  The hashes should be cached as individual, name-ordered files to disk.
	//
	//	The Request function can then be implemented as function that seeks-to and reads the proper block
	//  in the original file. If it needs to open the file, it can initialize it and keep a pointer.
	//  It should also open, read, and close the accompanying, pre-processed hash-file
	//  (this can be done concurrently).
	//  After writing the block to the response stream/buffer, the hash can be appended.
	//  The hashes are opened/closed as needed, but the media file can be kept open until the stream ends or times-out.
	//  This technique reduces the disk storage requirements, working-memory, and file open/close syscalls.

	if e.finished {
		finalBlock := e.nextBlock
		e.nextBlock = nil
		return finalBlock, io.EOF
	}

	// initial request
	if e.file == nil {
		fmt.Printf("[encoder] Opening %q\n", e.FileName)
		var err error
		e.file, err = os.Open(e.FileName)
		// there is no accompanying defer for this open file because it will be closed on the final read
		if err != nil {
			return nil, err
		}
	}

	// save and hash the current block to be returned
	returnBlock := append(e.nextBlock, sha256.New().Sum(e.nextBlock)...)

	// read and store the block for the next request
	if e.BlockSize <= 0 {
		fmt.Printf("[encoder] Warning: invalid BlockSize %q, defaulting to %q\n", e.BlockSize, defaultBlockSize)
		e.BlockSize = defaultBlockSize
	}
	e.nextBlock = make([]byte, e.BlockSize)
	_, err := e.file.Read(e.nextBlock)

	if err == io.EOF {
		// next Request will be the end of stream
		e.finished = true
		// don't return EOF on this request
		err = nil
		closeErr := e.file.Close()
		// ignore if there's an error closing the file, the stream must go on
		if closeErr != nil {
			fmt.Printf("[encoder] Warning: failed to close %q: %v\n", e.FileName, closeErr)
		}
	} else if err != nil {
		fmt.Printf("[encoder] Error: could not read block from file %q: %v\n", e.FileName, err)
	}

	// non-nil, non-EOF errors are returned to the client
	return returnBlock, err
}

// IsFinished reutrns whether the stream has been closed
func (e *Encoder) IsFinished() bool {
	return e.finished
}

// Close is a helper for the client to end the stream early
func (e *Encoder) Close() error {
	e.finished = true
	return e.file.Close()
}
