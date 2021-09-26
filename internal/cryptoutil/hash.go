package cryptoutil

import (
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
	"hash"
)

const (
	SHA2_256 = "SHA256"
	SHA3_256 = "SHA3_256"
)

type Hasher interface {
	Hash(msg []byte) []byte
	GetHash() hash.Hash
}

type hasher struct {
	hash func() hash.Hash
}

func (h hasher) Hash(msg []byte) []byte {
	hasher := h.GetHash()
	hasher.Write(msg)
	return hasher.Sum(nil)
}

func (h hasher) GetHash() hash.Hash {
	return h.hash()
}

func Hash(msg []byte, opt string) ([]byte, error) {
	var hasher hash.Hash
	switch opt {
	case SHA2_256:
		hasher = sha256.New()
	case SHA3_256:
		hasher = sha3.New256()
	default:
		return nil, errors.New("not support hash func")
	}
	hasher.Write(msg)
	return hasher.Sum(nil), nil
}

func ComputeSHA256(data []byte) (hash []byte) {
	var err error
	hash, err = Hash(data, SHA2_256)
	if err != nil {
		panic(fmt.Errorf("Failed computing SHA256 on [% x]", data))
	}
	return
}

// 获取 hash
func GetHash() hash.Hash {
	return sha256.New()
}
