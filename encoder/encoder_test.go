package encoder

import (
	"crypto/sha256"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestEncoder(t *testing.T) {
	type testCase struct {
		name       string
		e          Encoder
		hasInitial bool
		maxRequest int64
		enableLog  bool
	}
	table := []testCase{
		{
			name: "10blocks",
			e: Encoder{
				FileName:  "../testdata/test_0",
				BlockSize: 1024,
			},
			hasInitial: true,
			maxRequest: 10,
			enableLog:  true,
		},
		{
			name: "misaligned",
			e: Encoder{
				FileName:  "../testdata/test_1",
				BlockSize: 1024,
			},
			hasInitial: true,
			maxRequest: 11,
			enableLog:  true,
		},
		{
			name: "largerFile",
			e: Encoder{
				FileName:  "../testdata/test_01.input.mp4",
				BlockSize: 4096,
			},
			hasInitial: true,
			maxRequest: 5411,
		},
	}

	for i, tc := range table {
		t.Run(fmt.Sprintf("TestEncoder_%d_%s", i, tc.name), func(t *testing.T) {
			tc.e.PreProcess()

			// initial hash
			hash, err := tc.e.Request(0)
			if tc.enableLog {
				t.Log(0)
				t.Log(hash)
			}
			if tc.hasInitial && err != nil {
				t.Fatalf("failed on intialHash: %v", err)
			} else if !tc.hasInitial && err == nil {
				t.Fatalf("expecting err for intialHash, got: %v", err)
			}

			// all hashed blocks
			for i := int64(1); i < tc.maxRequest; i++ {
				blockHash, err := tc.e.Request(i)

				if tc.enableLog {
					t.Log(i)
					t.Log(blockHash)
				}

				if err != nil {
					t.Fatalf("failed on iteration %d: %v", i, err)
				}

				h := sha256.New()
				h.Write(blockHash)
				clientHash := h.Sum(nil)
				if !reflect.DeepEqual(hash, clientHash) {
					t.Fatalf("iteration %d, hashes do not match %v, %v", i, hash, clientHash)
				}

				// hash is stored for next block
				hash = blockHash[len(blockHash)-32:]
			}

			// final 0-hashed block
			blockHash, err := tc.e.Request(tc.maxRequest)

			if tc.enableLog {
				t.Log(tc.maxRequest)
				t.Log(blockHash)
			}

			if err != nil {
				t.Fatalf("failed on final block %d: %v", tc.maxRequest, err)
			}

			h := sha256.New()
			h.Write(blockHash)
			clientHash := h.Sum(nil)
			if !reflect.DeepEqual(hash, clientHash) {
				t.Fatalf("Final iteration: %d, hashes do not match %v, %v", tc.maxRequest, hash, clientHash)
			}

			// hash should be all 0's
			hash = blockHash[len(blockHash)-32:]
			if !reflect.DeepEqual(hash, make([]byte, 32)) {
				t.Fatalf("Final iteration: %d, trailing 0-hash expected, got: %v", tc.maxRequest, hash)
			}

			// subsequent requests all EOF /w nil blockHash
			for i := int64(tc.maxRequest + 1); i <= tc.maxRequest+6; i++ {
				blockHash, err := tc.e.Request(i)
				if tc.enableLog {
					t.Log(i)
					t.Log(blockHash)
				}
				if err != io.EOF {
					t.Fatalf("iteration %d expected %v, got: %v", i, io.EOF, err)
				}
				if blockHash != nil {
					t.Fatalf("iteration %d expected nil blockHash, got: %v", i, blockHash)
				}
			}
		})
	}
}
