package example

import (
	"context"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/gateway"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var (
	CLUSTER_NODE = ""
	CRYPTO_FILE_BASE = "./crypto-config"
)

var (
	org1 = organization {
		Name: "Org1",
		MspID: "Org1MSP",
		RootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/msp/cacerts/ca.org1.example.com-cert.pem"),
		TlsRootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/msp/tlscacerts/tlsca.org1.example.com-cert.pem"),
		Peers: []peer{
			{
				Url: "",
				Hostname: "peer0.org1.example.com",
			},
			{
				Url: "",
				Hostname: "peer1.org1.example.com",
			},
		},
		Admin: user{
			SignCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/signcerts/Admin@org1.example.com-cert.pem"),
			KeyCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/keystore/prov_sk"),
		},
	}

	org2 = organization{
		Name: "Org2",
		MspID: "Org1MSP",
		RootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/msp/cacerts/ca.org2.example.com-cert.pem"),
		TlsRootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/msp/tlscacerts/tlsca.org2.example.com-cert.pem"),
		Peers: []peer{
			{
				Url: "",
				Hostname: "peer0.org2.example.com",
			},
			{
				Url: "",
				Hostname: "peer1.org2.example.com",
			},
		},
		Admin: user{
			SignCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp/signcerts/Admin@org2.example.com-cert.pem"),
			KeyCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/keystore/prov_sk"),
		},
	}

	ordererOrg = organization{
		Name: "OrdererOrg",
		MspID: "OrdererMSP",
		RootCert: filepath.Join(CRYPTO_FILE_BASE, "ordererOrganizations/example.com/msp/cacerts/ca.example.com-cert.pem"),
		TlsRootCert: filepath.Join(CRYPTO_FILE_BASE, "ordererOrganizations/example.com/msp/tlscacerts/tlsca.example.com-cert.pem"),
		Peers: []peer{
			{
				Url: "",
				Hostname: "orderer0.example.com",
			},
			{
				Url: "",
				Hostname: "orderer1.example.com",
			},
			{
				Url: "",
				Hostname: "orderer2.example.com",
			},
		},
		Admin: user{
			SignCert: filepath.Join(CRYPTO_FILE_BASE, "ordererOrganizations/example.com/users/Admin@example.com/msp/signcerts/Admin@example.com-cert.pem"),
			KeyCert: filepath.Join(CRYPTO_FILE_BASE, "ordererOrganizations/example.com/users/Admin@example.com/msp/keystore/priv_sk"),
		},
	}
)

type peer struct {
	Url  string
	Hostname string
}

type user struct {
	SignCert string  // path to file
	KeyCert string  // path to file
}

type organization struct {
	Name  string
	MspID  string
	RootCert  string  // path to file
	TlsRootCert string // path to file
	Admin    user
	Peers []peer
}

type consortium struct {
	OrdererOrg  organization
	PeerOrgs    []organization
}

func (o organization) toProtoOrg() (*protoutil.Organization, error) {
	org := &protoutil.Organization{
		Name: o.Name,
		MspId: o.MspID,
	}
	rootCert, err := ioutil.ReadFile(o.RootCert)
	if err != nil {
		return nil, err
	}
	tlsRootCert, err := ioutil.ReadFile(o.TlsRootCert)
	if err != nil {
		return nil, err
	}

	org.RootCert = rootCert
	org.TlsRootCert = tlsRootCert

	return org, nil
}

func (o organization) getSigner() (*protoutil.Signer, error) {
	signer := &protoutil.Signer{MspId: o.MspID}
	signCert, err := ioutil.ReadFile(o.Admin.SignCert)
	if err != nil {
		return nil, err
	}
	keyCert, err := ioutil.ReadFile(o.Admin.KeyCert)
	if err != nil {
		return nil, err
	}
	signer.Cert= signCert
	signer.Key = keyCert
	return signer, nil
}

func (o organization) getPeers() ([]*protoutil.Peer, error) {
	var peers []*protoutil.Peer
	tlsRootCert, err := ioutil.ReadFile(o.TlsRootCert)
	if err != nil {
		return nil, err
	}
	for _, p := range o.Peers {
		peers = append(peers, &protoutil.Peer{
			Url: p.Url,
			HostName: p.Hostname,
			TlsRootCert: tlsRootCert,
		})
	}

	return peers, nil
}

func (o organization) getOrderer(idx int) (*protoutil.Orderer, error) {
	tlsRootCert, err := ioutil.ReadFile(o.TlsRootCert)
	if err != nil {
		return nil, err
	}

	return &protoutil.Orderer{
		Url: o.Peers[idx].Url,
		HostName: o.Peers[idx].Hostname,
		TlsRootCert: tlsRootCert,
	}, nil
}

func createChannel(channelID string, signer *protoutil.Signer, orderer organization, members []organization) error {
	req := &protoutil.CreateChannelRequest{
		ChannelId: channelID,
		ConsortiumName: "SampleConsortium",
		Signer: signer,
	}
	req.Orderer, _ = orderer.getOrderer(0)
	for _, m := range members {
		org, err := m.toProtoOrg()
		if err != nil {
			return err
		}
		req.Members = append(req.Members, org)
	}
	rsp, err := gateway.ChannelCreate(context.Background(), req)
	if err != nil {
		return err
	}

	if rsp.Status != gateway.RESPONSE_OK {
		return fmt.Errorf("fail to create channel with message: %v", rsp.Message)
	}
	return nil
}

func joinChannel(channelID string, org organization) error {
	signer, err := org.getSigner()
	if err != nil {
		return err
	}

	peers, err := org.getPeers()
	if err != nil {
		return err
	}
	req := &protoutil.JoinChannelRequest{
		ChannelId: channelID,
		Signer: signer,
		Peers: peers,
	}
	rsp, err := gateway.ChannelJoin(context.Background(), req)
	if err != nil {
		return err
	}

	if rsp.Status != gateway.RESPONSE_OK {
		return fmt.Errorf("fail to join channel:%v", rsp.Message)
	}
	return nil
}

func TestChannel_Create(t *testing.T) {
	c := consortium{
		PeerOrgs: []organization{org1, org2},
		OrdererOrg: ordererOrg,
	}
	channelID := "mychannel"
	signer, err := c.PeerOrgs[0].getSigner()
	if err != nil {
		t.Fatal(err)
	}
	err = createChannel(channelID, signer, c.OrdererOrg, c.PeerOrgs)
	if err != nil {
		t.Fatal(err)
	}

	channelMap := []struct {
		ID  string
		Members []organization
	}{
		{
			"channel01", []organization{org1,org2},
		},
		{
			"channel02", []organization{org1, org2},
		},
		{
			"channel03", []organization{org1},
		},
	}

	for _, cm := range channelMap {
		signer, err := cm.Members[0].getSigner()
		if err != nil {
			t.Fatal(err)
		}
		err = createChannel(cm.ID, signer, c.OrdererOrg, cm.Members)
		if err != nil {
			t.Fatal("fail ot create channel", cm.ID, "error:", err)
		}
	}
}