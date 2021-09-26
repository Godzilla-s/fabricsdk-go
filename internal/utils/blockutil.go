package utils

import (
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
)

// GetMetadataFromBlock retrieves metadata at the specified index.
func GetMetadataFromBlock(block *cb.Block, index cb.BlockMetadataIndex) (*cb.Metadata, error) {
	if block.Metadata == nil {
		return nil, errors.New("no metadata in block")
	}

	if len(block.Metadata.Metadata) <= int(index) {
		return nil, errors.Errorf("no metadata at index [%s]", index)
	}

	md := &cb.Metadata{}
	err := proto.Unmarshal(block.Metadata.Metadata[index], md)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling metadata at index [%s]", index)
	}
	return md, nil
}

// GetLastConfigIndexFromBlock retrieves the index of the last config block as
// encoded in the block metadata
func GetLastConfigIndexFromBlock(block *cb.Block) (uint64, error) {
	m, err := GetMetadataFromBlock(block, cb.BlockMetadataIndex_SIGNATURES)
	if err != nil {
		return 0, errors.WithMessage(err, "failed to retrieve metadata")
	}
	// TODO FAB-15864 Remove this fallback when we can stop supporting upgrade from pre-1.4.1 orderer
	if len(m.Value) == 0 {
		m, err := GetMetadataFromBlock(block, cb.BlockMetadataIndex_LAST_CONFIG)
		if err != nil {
			return 0, errors.WithMessage(err, "failed to retrieve metadata")
		}
		lc := &cb.LastConfig{}
		err = proto.Unmarshal(m.Value, lc)
		if err != nil {
			return 0, errors.Wrap(err, "error unmarshaling LastConfig")
		}
		return lc.Index, nil
	}

	obm := &cb.OrdererBlockMetadata{}
	err = proto.Unmarshal(m.Value, obm)
	if err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal orderer block metadata")
	}
	return obm.LastConfig.Index, nil
}

