package orderer

import (
	"context"
	"crypto/tls"
	"github.com/godzilla-s/fabricsdk-go/internal/client"
	"github.com/godzilla-s/fabricsdk-go/internal/comm"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	cb "github.com/hyperledger/fabric-protos-go/common"
	ab "github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/pkg/errors"
	"time"
)

type Config struct {
	Host  string
	ServiceOverrideName string
	RootTlsCert  []byte
}

type Client interface {
	GetAddress() string
	GetBroadcastClient() (BroadcastClient, error)
	GetDeliverClient(signer cryptoutil.Signer, channelID string, bestEffort bool) (OrdererDeliverClient, error)
}

type BroadcastClient interface {
	Send(env *cb.Envelope) error
	Close() error
}

type OrdererDeliverClient interface {
	GetSpecifiedBlock(num uint64) (*cb.Block, error)
	GetOldestBlock() (*cb.Block, error)
	GetNewestBlock() (*cb.Block, error)
	Close() error
}

type commonClient struct {
	*comm.GRPCClient
	address string
	sn      string
}

type ordererClient struct {
	*commonClient
}

func (oc *ordererClient) Broadcast() (ab.AtomicBroadcast_BroadcastClient, error) {
	conn, err := oc.commonClient.NewConnection(oc.address, comm.ServerNameOverride(oc.sn))
	if err != nil {
		return nil, errors.WithMessagef(err, "orderer client failed to connect to %s", oc.address)
	}
	return ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
}

func (oc *ordererClient) Deliver() (ab.AtomicBroadcast_DeliverClient, error) {
	conn, err := oc.commonClient.NewConnection(oc.address, comm.ServerNameOverride(oc.sn))
	if err != nil {
		return nil, errors.WithMessagef(err, "orderer client failed to connect to %s", oc.address)
	}
	return ab.NewAtomicBroadcastClient(conn).Deliver(context.TODO())
}

func (oc *ordererClient) GetBroadcastClient() (BroadcastClient, error) {
	bc, err := oc.Broadcast()
	if err != nil {
		return nil, err
	}
	return &broadcastClient{bc}, nil
}

func (oc *ordererClient) GetAddress() string {
	return oc.sn
}

// Certificate returns the TLS client certificate (if available)
func (oc *ordererClient) Certificate() tls.Certificate {
	return oc.commonClient.Certificate()
}

func (oc *ordererClient) GetDeliverClient(signer cryptoutil.Signer, channelID string, bestEffort bool) (OrdererDeliverClient, error) {
	deliver, err := oc.Deliver()
	if err != nil {
		return nil, err
	}

	var tlsCertHash []byte
	if len(oc.Certificate().Certificate) > 0 {
		tlsCertHash = cryptoutil.ComputeSHA256(oc.Certificate().Certificate[0])
	}
	ds := DeliverService{
		Client: deliver,
		Signer: signer,
		TLSCertHash: tlsCertHash,
		ChannelID: channelID,
		BestEffort: bestEffort,
	}
	return &ds, nil
}

func New(url, serviceName string, tlsRootCert []byte) (Client, error) {
	config := Config{Host: url, ServiceOverrideName: serviceName, RootTlsCert: tlsRootCert}
	return config.New()
}

func (c *Config) New(opts ...client.Option) (Client, error) {
	config := &comm.ClientConfig{}
	secOpts := comm.SecureOptions{}
	if c.RootTlsCert != nil {
		secOpts.UseTLS = true
		secOpts.ServerRootCAs = [][]byte{c.RootTlsCert}
	}
	config.Timeout = 3 * time.Second
	config.SecOpts = secOpts
	for _, opt := range opts {
		config = opt(config)
	}
	gClient, err := comm.NewGRPCClient(*config)
	if err != nil {
		return nil, err
	}
	return &ordererClient{
		commonClient: &commonClient{
			GRPCClient: gClient,
			sn: c.ServiceOverrideName,
			address: c.Host,
		},
	}, nil
}


type broadcastClient struct {
	client ab.AtomicBroadcast_BroadcastClient
}

func (bc *broadcastClient) getAck() error {
	msg, err := bc.client.Recv()
	if err != nil {
		return err
	}
	if msg.Status != cb.Status_SUCCESS {
		return errors.Errorf("got unexpected status: %v -- %s", msg.Status, msg.Info)
	}
	return nil
}

func (bc *broadcastClient) Send(env *cb.Envelope) error {
	err := bc.client.Send(env)
	if err != nil {
		return errors.WithMessage(err, "could not send")
	}
	err = bc.getAck()
	return err
}

func (bc *broadcastClient) Close() error {
	return bc.client.CloseSend()
}
