package main

import (
	"fmt"
	"io"
	"os"

	"stealthybox.dev/go-hash-player/decoder"
	"stealthybox.dev/go-hash-player/encoder"
)

func main() {
	Stream("testdata/test_0", "out_0")
	Stream("testdata/test_1", "out_1")
	Stream("testdata/test_01.input.mp4", "out_01.mp4")
}

func Stream(infile, outfile string) {
	e := encoder.Encoder{
		FileName: infile,
	}

	err := e.PreProcess()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// quick way to ensure our file is empty, ignore removeErr
	_ = os.Remove(outfile)
	f, fErr := os.OpenFile(outfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if fErr != nil {
		panic("Failed opening outfile: " + outfile)
	}
	defer f.Close()

	hash, reqErr := e.Request(0)
	var decodeErr error

	for i := int64(1); reqErr == nil && decodeErr == nil; i++ {
		hashedBlock, reqErr := e.Request(i)
		if reqErr != nil {
			break
		}

		var block []byte
		block, hash, decodeErr = decoder.Decode(hash, hashedBlock)
		if decodeErr != nil {
			break
		}

		_, fErr := f.Write(block)
		if fErr != nil {
			panic("Failed writing block to outfile: " + outfile)
		}
	}

	if reqErr == io.EOF {
		fmt.Println("Success: end of stream")
	} else if reqErr != nil {
		fmt.Printf("Error: %v\n", reqErr)
	}

	if decodeErr != nil {
		fmt.Printf("Error: %v\n", reqErr)
	}
}
