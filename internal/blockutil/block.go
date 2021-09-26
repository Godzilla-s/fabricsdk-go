package blockutil

import (
	"encoding/asn1"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/blockutil/block"
	"github.com/godzilla-s/fabricsdk-go/internal/blockutil/config"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
	"math"
)

type asn1Header struct {
	Number       int64
	PreviousHash []byte
	DataHash     []byte
}

// GetHash 获取区块哈希
func GetHash(b *cb.Block) ([]byte, error) {
	asn1Header := asn1Header{
		PreviousHash: b.Header.PreviousHash,
		DataHash:     b.Header.DataHash,
	}
	if b.Header.Number > uint64(math.MaxInt64) {
		panic(fmt.Errorf("Golang does not currently support encoding uint64 to asn1"))
	} else {
		asn1Header.Number = int64(b.Header.Number)
	}

	result, err := asn1.Marshal(asn1Header)
	if err != nil {
		panic(err)
	}
	return cryptoutil.Hash(result, cryptoutil.SHA2_256)
}

// GetHashOrPanic 获取区块哈希
func GetHashOrPanic(b *cb.Block) []byte {
	hash, err := GetHash(b)
	if err != nil {
		panic(err)
	}
	return hash
}

func UnmarshalConfig(b *cb.Block) (*config.Block, error) {
	block := &config.Block{
		Header: b.Header,
		Metadata: b.Metadata,
		Data: make([]*config.Envelope, len(b.Data.Data)),
	}
	for i, data := range b.Data.Data {
		env, err := utils.UnmarshalEnvelope(data)
		if err != nil {
			return nil, errors.WithMessagef(err, "unmarshal envelope")
		}

		payload, err := config.UnmarshalConfigPayload(env.Payload)
		if err != nil {
			return nil, errors.WithMessage(err, "unmarshal config payload")
		}

		envelope := config.Envelope{
			Payload: payload,
			Signature: env.Signature,
		}
		block.Data[i] = &envelope
	}

	return block, nil
}

func UnmarshalBlock(b *cb.Block) (*block.Block, error) {
	blk := &block.Block{
		Header:   b.Header,
		Metadata: b.Metadata,
		Data:     make([]*block.Envelope, len(b.Data.Data)),
	}

	for i, data := range b.Data.Data {
		env, err := utils.UnmarshalEnvelope(data)
		if err != nil {
			return nil, errors.WithMessagef(err, "unmarshal envelope")
		}
		envelope, err := block.UnmarshalEnvelope(env)
		if err != nil {
			return nil, errors.WithMessage(err, "unmarshal envelope")
		}
		blk.Data[i] = envelope
	}

	return blk, nil
}
