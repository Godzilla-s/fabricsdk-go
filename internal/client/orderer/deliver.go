package orderer

import (
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	cb "github.com/hyperledger/fabric-protos-go/common"
	ab "github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/pkg/errors"
)

var (
	seekNewest = &ab.SeekPosition{
		Type: &ab.SeekPosition_Newest{
			Newest: &ab.SeekNewest{},
		},
	}
	seekOldest = &ab.SeekPosition{
		Type: &ab.SeekPosition_Oldest{
			Oldest: &ab.SeekOldest{},
		},
	}
)

type DeliverService struct {
	Signer      cryptoutil.Signer
	Client      ab.AtomicBroadcast_DeliverClient
	ChannelID   string
	TLSCertHash []byte
	BestEffort  bool
}

func (ds *DeliverService) seekSpecified(blockNum uint64) error {
	seekPosition := &ab.SeekPosition{
		Type: &ab.SeekPosition_Specified{
			Specified: &ab.SeekSpecified{
				Number: blockNum,
			},
		},
	}
	env := seekHelper(ds.ChannelID, seekPosition, ds.TLSCertHash, ds.Signer, ds.BestEffort)
	return ds.Client.Send(env)
}

func (ds *DeliverService) seekOldest() error {
	env := seekHelper(ds.ChannelID, seekOldest, ds.TLSCertHash, ds.Signer, ds.BestEffort)
	return ds.Client.Send(env)
}

func (ds *DeliverService) seekNewest() error {
	env := seekHelper(ds.ChannelID, seekNewest, ds.TLSCertHash, ds.Signer, ds.BestEffort)
	return ds.Client.Send(env)
}

func (ds *DeliverService) readBlock() (*cb.Block, error) {
	msg, err := ds.Client.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "error receiving")
	}

	switch t := msg.Type.(type) {
	case *ab.DeliverResponse_Status:
		return nil,errors.Errorf("can't read the block: %v", t)
	case *ab.DeliverResponse_Block:
		if resp, err := ds.Client.Recv(); err != nil { // Flush the success message
			// TODO
		} else if status := resp.GetStatus(); status != cb.Status_SUCCESS {
			// TODO
		}
		return t.Block, nil
	default:
		return nil, errors.Errorf("response error: unknown type %T", t)
	}
}

// GetSpecifiedBlock gets the specified block from a peer/orderer's deliver
// service
func (ds *DeliverService) GetSpecifiedBlock(num uint64) (*cb.Block, error) {
	err := ds.seekSpecified(num)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting specified block")
	}

	return ds.readBlock()
}

// GetOldestBlock gets the oldest block from a peer/orderer's deliver service
func (ds *DeliverService) GetOldestBlock() (*cb.Block, error) {
	err := ds.seekOldest()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting oldest block")
	}

	return ds.readBlock()
}

// GetNewestBlock gets the newest block from a peer/orderer's deliver service
func (ds *DeliverService) GetNewestBlock() (*cb.Block, error) {
	err := ds.seekNewest()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting newest block")
	}

	return ds.readBlock()
}

// Close closes a deliver client's connection
func (ds *DeliverService) Close() error {
	return ds.Client.CloseSend()
}

func seekHelper(
	channelID string,
	position *ab.SeekPosition,
	tlsCertHash []byte,
	signer cryptoutil.Signer,
	bestEffort bool,
) *cb.Envelope {
	seekInfo := &ab.SeekInfo{
		Start:    position,
		Stop:     position,
		Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
	}

	if bestEffort {
		seekInfo.ErrorResponse = ab.SeekInfo_BEST_EFFORT
	}
	env, err := utils.CreateSignedEnvelopeWithTLSBinding(
		cb.HeaderType_DELIVER_SEEK_INFO,
		channelID,
		signer,
		seekInfo,
		int32(0),
		uint64(0),
		tlsCertHash,
	)
	if err != nil {
		return nil
	}
	return env
}
