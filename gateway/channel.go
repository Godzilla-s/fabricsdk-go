package gateway

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/channel"
	"github.com/godzilla-s/fabricsdk-go/internal/client/orderer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/pkg/errors"
)

func CreateChannel(ctx context.Context, req *protoutil.CreateChannelRequest) (*protoutil.Response, error)  {
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

func JoinChannel(ctx context.Context, req *protoutil.JoinChannelRequest) (*protoutil.Response, error) {
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
	responses, err := channel.Join(signer, pClients, oClient, req.ChannelId)
	if err != nil {
		return nil, errors.WithMessage(err, "join peer")
	}

	_ = responses
	return nil, nil
}

func UpdateChannel(ctx context.Context, req *protoutil.UpdateChannelRequest) (*protoutil.Response, error) {
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
