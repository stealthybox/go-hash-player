package encoder

import (
	"fmt"
	"io"
	"testing"
)

func TestEncoder0(t *testing.T) {
	e := Encoder{
		FileName:  "../testdata/test_0",
		BlockSize: 1024,
	}

	// initial and intermediate blocks
	for i := 0; i < 11; i++ {
		blockHash, err := e.Request()
		fmt.Println(blockHash)
		if err != nil {
			t.Fatalf("failed on iteration %v: %v", i, err)
		}
	}

	// final block
	blockHash, err := e.Request()
	fmt.Println(blockHash)
	if err != io.EOF {
		t.Fatalf("final block: expected `io.EOF`, got: %v", err)
	}

	// additional requests return <nil, io.EOF>
	nilBlock, err := e.Request()
	fmt.Println(nilBlock)
	if err != io.EOF {
		t.Fatalf("nil block: expected `io.EOF`, got: %v", err)
	}
	if nilBlock != nil {
		t.Fatalf("nil block: expected `nil`, got: %v", nilBlock)
	}

}
