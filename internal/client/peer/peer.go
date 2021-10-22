package peer

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/client"
	"github.com/godzilla-s/fabricsdk-go/internal/comm"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"time"
)

type Client interface {
	GetDeliverService() (DeliverClient, error)
	GetEndorser() (pb.EndorserClient, error)
	GetDeliverClient() (pb.DeliverClient, error)
	GetCertificate() tls.Certificate
	GetAddress() string
}

type Config struct {
	Host  string
	ServiceOverrideName string
	RootTlsCert  []byte
}

type DeliverClient interface {
	Send(env *cb.Envelope) error
	Recv() (*pb.DeliverResponse, error)
}

type peerDeliverService struct {
	client pb.Deliver_DeliverClient
}

func (ds *peerDeliverService) Send(env *cb.Envelope) error {
	ds.client.Send(env)
	return nil
}

func (ds *peerDeliverService) Recv() (*pb.DeliverResponse, error) {
	return ds.client.Recv()
}

func New(url, serviceName string, tlsRootCert []byte, opts ...client.Option) (Client, error) {
	config := Config{Host: url, ServiceOverrideName: serviceName, RootTlsCert: tlsRootCert}
	return config.New(opts...)
}

func (c *Config) New(opts ...client.Option) (Client, error) {
	clientConfig := &comm.ClientConfig{}
	secOpts := comm.SecureOptions{}
	if c.RootTlsCert != nil {
		secOpts.UseTLS = true
		secOpts.ServerRootCAs = [][]byte{c.RootTlsCert}
	}
	clientConfig.SecOpts = secOpts
	for _, opt := range opts {
		clientConfig = opt(clientConfig)
	}
	if clientConfig.Timeout == 0 {
		clientConfig.Timeout = 3 * time.Second
	}
	gClient, err := comm.NewGRPCClient(*clientConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to create grpc client")
	}
	pClient := &peerClient{&commonClient{
		GRPCClient: gClient,
		address:    c.Host,
		sn:         c.ServiceOverrideName,
	}}
	return pClient, nil
}


type commonClient struct {
	*comm.GRPCClient
	address string
	sn      string
}

type peerClient struct {
	*commonClient
}


func (pc *peerClient) GetEndorser() (pb.EndorserClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, comm.ServerNameOverride(pc.sn))
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("endorser client failed to connect to %s, %s", pc.address, pc.sn))
	}

	return pb.NewEndorserClient(conn), nil
}

func (pc *peerClient) Deliver() (pb.Deliver_DeliverClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, comm.ServerNameOverride(pc.sn))
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("endorser client failed to connect to %s", pc.address))
	}
	//return pb.NewDeliverClient(conn).Deliver(context.TODO())
	return pb.NewDeliverClient(conn).Deliver(context.Background())
}

// PeerDeliver returns a client for the Deliver service for peer-specific use
// cases (i.e. DeliverFiltered)
func (pc *peerClient) GetDeliverClient() (pb.DeliverClient, error) {
	conn, err := pc.commonClient.NewConnection(pc.address, comm.ServerNameOverride(pc.sn))
	if err != nil {
		return nil, errors.WithMessagef(err, "deliver client failed to connect to %s", pc.address)
	}
	return pb.NewDeliverClient(conn), nil
}

// Certificate returns the TLS client certificate (if available)
func (pc *peerClient) GetCertificate() tls.Certificate {
	return pc.commonClient.Certificate()
}

func (pc *peerClient) GetDeliverService() (DeliverClient, error){
	dc, err := pc.Deliver()
	if err != nil {
		return nil, err
	}
	return &peerDeliverService{
		client: dc,
	}, nil
}

func (pc *peerClient) GetAddress() string {
	return pc.sn
}
