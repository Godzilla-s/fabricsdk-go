package gateway

import (
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/chaincode"
	"github.com/godzilla-s/fabricsdk-go/internal/channel"
	orderercli "github.com/godzilla-s/fabricsdk-go/internal/client/orderer"
	peercli "github.com/godzilla-s/fabricsdk-go/internal/client/peer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/hyperledger/fabric-protos-go/peer"
)

const (
	RESPONSE_OK = 200
	RESPONSE_FAIL = 500
)

func createSigner(signer *protoutil.Signer) (cryptoutil.Signer, error) {
	cs, err := cryptoutil.GetMyCryptoSuiteFromBytes(signer.Key, signer.Cert, signer.MspId)
	if err != nil {
		return nil, err
	}
	return cs.NewSigner()
}


func createPeerClients(peers []*protoutil.Peer) ([]peercli.Client, error) {
	pClients := make([]peercli.Client, len(peers))
	for i, p := range peers {
		peerCli, err := peercli.New(p.Url, p.HostName, p.TlsRootCert)
		if err != nil {
			return nil, err
		}
		pClients[i] = peerCli
	}
	return pClients, nil
}

func createCommonFactory(commiter *protoutil.Peer, endorsers []*protoutil.Peer, ord *protoutil.Orderer) (*chaincode.CommonFactory, error) {
	cf := &chaincode.CommonFactory{}
	commitCli, err := peercli.New(commiter.Url, commiter.HostName, commiter.TlsRootCert)
	if err != nil {
		return nil, err
	}
	ordererCli, err := orderercli.New(ord.Url, ord.HostName, ord.TlsRootCert)
	if err != nil {
		return nil, err
	}
	cf.Committer, err = commitCli.GetEndorser()
	cf.TLSCert = commitCli.GetCertificate()
	if err != nil {
		return nil, err
	}
	cf.OClient = ordererCli
	cf.Endorsers = make([]peer.EndorserClient, len(endorsers))
	cf.PeerAddresses = make([]string, len(endorsers))
	endorserClients, err := createPeerClients(endorsers)
	if err != nil {
		return nil, err
	}
	for i, e := range endorserClients {
		endorser, err := e.GetEndorser()
		if err != nil {
			return nil, err
		}
		cf.Endorsers[i] = endorser
		deliver, err := e.GetDeliverClient()
		if err != nil {
			return nil, err
		}
		cf.Delivers[i] = deliver
		cf.PeerAddresses[i] = e.GetAddress()
	}
	return cf, nil
}

func createChannelOrg(org *protoutil.Organization) channel.Organization {
	rootCA, _ := cryptoutil.GetCertFromPEM(org.RootCert)
	tlsRootCA, _ := cryptoutil.GetCertFromPEM(org.TlsRootCert)
	return channel.Organization{
		Name: org.Name,
		ID: org.MspId,
		Type: org.Type,
		RootCA: rootCA,
		TLSRootCA: tlsRootCA,
	}
}