package gateway

import (
	"context"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/channel"
	"github.com/godzilla-s/fabricsdk-go/internal/client/orderer"
	"github.com/godzilla-s/fabricsdk-go/internal/client/peer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// ChannelCreate is interface to create channel in fabric network
func ChannelCreate(ctx context.Context, req *protoutil.CreateChannelRequest) (*protoutil.Response, error)  {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, errors.WithMessage(err, "create signer")
	}

	oClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, err
	}

	channelOrgs := make([]channel.Organization, len(req.Members))
	for i, org := range req.Members {
		rootCA, _ := cryptoutil.GetCertFromPEM(org.RootCert)
		tlsRootCA,  _ := cryptoutil.GetCertFromPEM(org.TlsRootCert)
		channelOrgs[i] = channel.Organization{
			Name: org.Name,
			ID: org.MspId,
			Type: org.Type,
			RootCA: rootCA,
			TLSRootCA: tlsRootCA,
		}
	}

	channelEnvelope, err := channel.CreateApplicationChannel(req.ChannelId, req.ConsortiumName, channelOrgs)
	if err != nil {
		return nil, errors.WithMessage(err, "create application channel")
	}

	err = channel.Create(signer, channelEnvelope, oClient)
	if err != nil {
		return &protoutil.Response{Status: 500, Message: err.Error()}, nil
	}
	return &protoutil.Response{Status: 200}, nil
}

// ChannelJoin is API for peer to join to channel
func ChannelJoin(ctx context.Context, req *protoutil.JoinChannelRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, errors.WithMessage(err, "create signer")
	}
	oClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, errors.WithMessage(err, "create orderer client")
	}
	pClients, err := createPeerClients(req.Peers)
	if err != nil {
		return nil, err
	}

	rsp, err := channel.Join2(signer, pClients[0], oClient, req.ChannelId)
	if err != nil {
		return nil, errors.WithMessage(err, "join peer")
	}

	return &protoutil.Response{
		Status: rsp.Status,
		Message: rsp.Message,
		Payload: rsp.Payload,
	}, nil
}

// ChannelUpdate is API for update channel config
func ChannelUpdate(ctx context.Context, req *protoutil.UpdateChannelRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}

	oClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, err
	}

	updateEnvelope := channel.NewChannelFromBytes(req.ChannelId, req.UpdateEnvelope)
	err = channel.Update(signer, updateEnvelope, oClient)
	if err != nil {
		return nil, err
	}
	return &protoutil.Response{Status: 200}, nil
}

// ChannelList is API to list channel of peer node
func ChannelList(ctx context.Context, req *protoutil.ListChannelsRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}
	pClient, err := peer.New(req.Peer.Url, req.Peer.HostName, req.Peer.TlsRootCert)
	if err != nil {
		return nil, err
	}
	channelRsp, err := channel.List(signer, pClient)
	if err != nil {
		return nil, err
	}
	resp := &protoutil.Response{}
	var channels []string

	for _, c := range channelRsp.Channels {
		channels = append(channels, c.ChannelId)
	}

	fmt.Println("channel:", channels)

	//resp.Payload, _ = proto.Marshal(channels)
	resp.Status = 200
	return resp, nil
}

func FetchBlock(ctx context.Context, req *protoutil.FetchBlockRequest) (*protoutil.Response, error) {
	// TODO
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}
	oClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, err
	}
	block, err := channel.FetchBlock(signer, oClient, req.ChannelId, uint64(req.Height))
	if err != nil {
		return nil, err
	}

	blockBytes, err := proto.Marshal(block)

	if err != nil {
		return nil, err
	}
	return &protoutil.Response{Payload: blockBytes, Status: 200}, nil
}

func FetchConfig(ctx context.Context, req *protoutil.FetchConfigRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}
	oClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, err
	}
	config, err := channel.FetchConfig(signer, oClient, req.ChannelId)
	if err != nil {
		return nil, err
	}

	resp := &protoutil.Response{}
	resp.Payload, err = proto.Marshal(config)
	if err != nil {
		return nil, err
	}
	resp.Status = 200
	return resp, nil
}

