package utils

import (
	"encoding/binary"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// ConcatenateBytes is useful for combining multiple arrays of bytes, especially for
// signatures or digests over multiple fields
func ConcatenateBytes(data ...[]byte) []byte {
	finalLength := 0
	for _, slice := range data {
		finalLength += len(slice)
	}
	result := make([]byte, finalLength)
	last := 0
	for _, slice := range data {
		for i := range slice {
			result[i+last] = slice[i]
		}
		last += len(slice)
	}
	return result
}

// EncodeOrderPreservingVarUint64 returns a byte-representation for a uint64 number such that
// all zero-bits starting bytes are trimmed in order to reduce the length of the array
// For preserving the order in a default bytes-comparison, first byte contains the number of remaining bytes.
// The presence of first byte also allows to use the returned bytes as part of other larger byte array such as a
// composite-key representation in db
func EncodeOrderPreservingVarUint64(number uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, number)
	startingIndex := 0
	size := 0
	for i, b := range bytes {
		if b != 0x00 {
			startingIndex = i
			size = 8 - i
			break
		}
	}
	sizeBytes := proto.EncodeVarint(uint64(size))
	if len(sizeBytes) > 1 {
		panic(fmt.Errorf("[]sizeBytes should not be more than one byte because the max number it needs to hold is 8. size=%d", size))
	}
	encodedBytes := make([]byte, size+1)
	encodedBytes[0] = sizeBytes[0]
	copy(encodedBytes[1:], bytes[startingIndex:])
	return encodedBytes
}

// DecodeOrderPreservingVarUint64 decodes the number from the bytes obtained from method 'EncodeOrderPreservingVarUint64'.
// It returns the decoded number, the number of bytes that are consumed in the process, and an error if the input bytes are invalid.
func DecodeOrderPreservingVarUint64(bytes []byte) (uint64, int, error) {
	s, numBytes := proto.DecodeVarint(bytes)

	switch {
	case numBytes != 1:
		return 0, 0, errors.Errorf("number of consumed bytes from DecodeVarint is invalid, expected 1, but got %d", numBytes)
	case s > 8:
		return 0, 0, errors.Errorf("decoded size from DecodeVarint is invalid, expected <=8, but got %d", s)
	case int(s) > len(bytes)-1:
		return 0, 0, errors.Errorf("decoded size (%d) from DecodeVarint is more than available bytes (%d)", s, len(bytes)-1)
	default:
		// no error
		size := int(s)
		decodedBytes := make([]byte, 8)
		copy(decodedBytes[8-size:], bytes[1:size+1])
		numBytesConsumed := size + 1
		return binary.BigEndian.Uint64(decodedBytes), numBytesConsumed, nil
	}
}

// ComputeStringHash computes the hash of the given string
func ComputeStringHash(input string) []byte {
	return ComputeHash([]byte(input))
}

// ComputeHash computes the hash of the given bytes
func ComputeHash(input []byte) []byte {
	return ComputeSHA256(input)
}

// ComputeSHA256 returns SHA2-256 on data
func ComputeSHA256(data []byte) (hash []byte) {
	hash, err := cryptoutil.Hash(data, cryptoutil.SHA2_256)
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA256 on [% x]", data))
	}
	return
}

// ComputeSHA3256 returns SHA3-256 on data
func ComputeSHA3256(data []byte) (hash []byte) {
	hash, err := cryptoutil.Hash(data, cryptoutil.SHA3_256)
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA3_256 on [% x]", data))
	}
	return
}
