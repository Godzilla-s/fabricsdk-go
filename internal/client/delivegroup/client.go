package delivegroup

import (
	"context"
	"crypto/tls"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	cb "github.com/hyperledger/fabric-protos-go/common"
	ab "github.com/hyperledger/fabric-protos-go/orderer"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"math"
	"sync"
)

// DeliverGroup holds all of the information needed to connect
// to a set of peers to wait for the interested txid to be
// committed to the ledgers of all peers. This functionality
// is currently implemented via the peer's DeliverFiltered service.
// An error from any of the peers/deliver clients will result in
// the invoke command returning an error. Only the first error that
// occurs will be set
type DeliverGroup struct {
	Clients     []*DeliverClient
	Certificate tls.Certificate
	ChannelID   string
	TxID        string
	Signer      cryptoutil.Signer
	mutex       sync.Mutex
	Error       error
	wg          sync.WaitGroup
}

// DeliverClient holds the client/connection related to a specific
// peer. The address is included for logging purposes
type DeliverClient struct {
	Client     pb.DeliverClient
	Connection pb.Deliver_DeliverClient
	Address    string
}

func NewDeliverGroup(
	deliverClients []pb.DeliverClient,
	peerAddresses []string,
	signer cryptoutil.Signer,
	certificate tls.Certificate,
	channelID string,
	txid string) *DeliverGroup {

	clients := make([]*DeliverClient, len(deliverClients))
	for i, client := range deliverClients {
		dc := &DeliverClient{
			Client:  client,
			Address: peerAddresses[i],
		}
		clients[i] = dc
	}

	dg := &DeliverGroup{
		Clients:     clients,
		Certificate: certificate,
		ChannelID:   channelID,
		TxID:        txid,
		Signer:      signer,
	}
	return dg
}

// Connect waits for all deliver clients in the group to connect to
// the peer's deliver service, receive an error, or for the context
// to timeout. An error will be returned whenever even a single
// deliver client fails to connect to its peer
func (dg *DeliverGroup) Connect(ctx context.Context) error {
	dg.wg.Add(len(dg.Clients))
	for _, client := range dg.Clients {
		go dg.ClientConnect(ctx, client)
	}
	readyCh := make(chan struct{})
	go dg.WaitForWG(readyCh)

	select {
	case <-readyCh:
		if dg.Error != nil {
			err := errors.WithMessage(dg.Error, "failed to connect to deliver on all peers")
			return err
		}
	case <-ctx.Done():
		err := errors.New("timed out waiting for connection to deliver on all peers")
		return err
	}

	return nil
}

// ClientConnect sends a deliver seek info envelope using the
// provided deliver client, setting the deliverGroup's Error
// field upon any error
func (dg *DeliverGroup) ClientConnect(ctx context.Context, dc *DeliverClient) {
	defer dg.wg.Done()
	df, err := dc.Client.DeliverFiltered(ctx)
	if err != nil {
		err = errors.WithMessagef(err, "error connecting to deliver filtered at %s", dc.Address)
		dg.setError(err)
		return
	}
	defer df.CloseSend()
	dc.Connection = df

	envelope := createDeliverEnvelope(dg.ChannelID, dg.Certificate, dg.Signer)
	err = df.Send(envelope)
	if err != nil {
		err = errors.WithMessagef(err, "error sending deliver seek info envelope to %s", dc.Address)
		dg.setError(err)
		return
	}
}

// Wait waits for all deliver client connections in the group to
// either receive a block with the txid, an error, or for the
// context to timeout
func (dg *DeliverGroup) Wait(ctx context.Context) error {
	if len(dg.Clients) == 0 {
		return nil
	}
	dg.wg.Add(len(dg.Clients))
	for _, client := range dg.Clients {
		go dg.ClientWait(client)
	}
	readyCh := make(chan struct{})
	go dg.WaitForWG(readyCh)

	select {
	case <-readyCh:
		if dg.Error != nil {
			return dg.Error
		}
	case <-ctx.Done():
		err := errors.New("timed out waiting for txid on all peers")
		return err
	}

	return nil
}

// ClientWait waits for the specified deliver client to receive
// a block eventclient with the requested txid
func (dg *DeliverGroup) ClientWait(dc *DeliverClient) {
	defer dg.wg.Done()
	for {
		resp, err := dc.Connection.Recv()
		if err != nil {
			err = errors.WithMessagef(err, "error receiving from deliver filtered at %s", dc.Address)
			dg.setError(err)
			return
		}
		switch r := resp.Type.(type) {
		case *pb.DeliverResponse_FilteredBlock:
			filteredTransactions := r.FilteredBlock.FilteredTransactions
			for _, tx := range filteredTransactions {
				if tx.Txid == dg.TxID {
					//logger.Infof("txid [%s] committed with status (%s) at %s", dg.TxID, tx.TxValidationCode, dc.Address)
					if tx.TxValidationCode != pb.TxValidationCode_VALID {
						err = errors.Errorf("transaction invalidated with status (%s)", tx.TxValidationCode)
						dg.setError(err)
					}
					return
				}
			}
		case *pb.DeliverResponse_Status:
			err = errors.Errorf("deliver completed with status (%s) before txid received", r.Status)
			dg.setError(err)
			return
		default:
			err = errors.Errorf("received unexpected response type (%T) from %s", r, dc.Address)
			dg.setError(err)
			return
		}
	}
}

// WaitForWG waits for the deliverGroup's wait group and closes
// the channel when ready
func (dg *DeliverGroup) WaitForWG(readyCh chan struct{}) {
	dg.wg.Wait()
	close(readyCh)
}

// setError serializes an error for the deliverGroup
func (dg *DeliverGroup) setError(err error) {
	dg.mutex.Lock()
	dg.Error = err
	dg.mutex.Unlock()
}

func createDeliverEnvelope(
	channelID string,
	certificate tls.Certificate,
	signer cryptoutil.Signer,
) *cb.Envelope {
	var tlsCertHash []byte
	// check for client certificate and create hash if present
	if len(certificate.Certificate) > 0 {
		tlsCertHash = cryptoutil.ComputeSHA256(certificate.Certificate[0])
	}

	start := &ab.SeekPosition{
		Type: &ab.SeekPosition_Newest{
			Newest: &ab.SeekNewest{},
		},
	}

	stop := &ab.SeekPosition{
		Type: &ab.SeekPosition_Specified{
			Specified: &ab.SeekSpecified{
				Number: math.MaxUint64,
			},
		},
	}

	seekInfo := &ab.SeekInfo{
		Start:    start,
		Stop:     stop,
		Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
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
		//logger.Errorf("Error signing envelope: %s", err)
		return nil
	}

	return env
}
