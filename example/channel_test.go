package example

import (
	"context"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/gateway"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

var (
	CLUSTER_NODE = ""
	CRYPTO_FILE_BASE = "D:\\test\\crypto-config"
	FABRIC_CLUSTER_ADDR = "192.168.1.103"
)

var (
	org1 = organization {
		Name: "org1",
		MspID: "Org1MSP",
		RootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/msp/cacerts/ca.org1.example.com-cert.pem"),
		TlsRootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/msp/tlscacerts/tlsca.org1.example.com-cert.pem"),
		Peers: []peer{
			{
				Url: FABRIC_CLUSTER_ADDR+":"+"7051",
				Hostname: "peer0.org1.example.com",
			},
			{
				Url: "",
				Hostname: "peer1.org1.example.com",
			},
		},
		Admin: user{
			SignCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/signcerts/Admin@org1.example.com-cert.pem"),
			KeyCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/keystore/priv_sk"),
		},
	}

	org2 = organization{
		Name: "org2",
		MspID: "Org2MSP",
		RootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/msp/cacerts/ca.org2.example.com-cert.pem"),
		TlsRootCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/msp/tlscacerts/tlsca.org2.example.com-cert.pem"),
		Peers: []peer{
			{
				Url: FABRIC_CLUSTER_ADDR+":"+"8051",
				Hostname: "peer0.org2.example.com",
			},
			{
				Url: "",
				Hostname: "peer1.org2.example.com",
			},
		},
		Admin: user{
			SignCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp/signcerts/Admin@org2.example.com-cert.pem"),
			KeyCert: filepath.Join(CRYPTO_FILE_BASE, "peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp/keystore/priv_sk"),
		},
	}

	ordererOrg = organization{
		Name: "Orderer",
		MspID: "OrdererMSP",
		RootCert: filepath.Join(CRYPTO_FILE_BASE, "ordererOrganizations/orderer.example.com/msp/cacerts/ca.orderer.example.com-cert.pem"),
		TlsRootCert: filepath.Join(CRYPTO_FILE_BASE, "ordererOrganizations/orderer.example.com/msp/tlscacerts/tlsca.orderer.example.com-cert.pem"),
		Peers: []peer{
			{
				Url: FABRIC_CLUSTER_ADDR+":7050",
				Hostname: "orderer0.orderer.example.com",
			},
			{
				Url: FABRIC_CLUSTER_ADDR+":7150",
				Hostname: "orderer1.orderer.example.com",
			},
			{
				Url: FABRIC_CLUSTER_ADDR+":7250",
				Hostname: "orderer2.orderer.example.com",
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

func (o organization) getListChannelRequest() (*protoutil.ListChannelsRequest, error) {
	var err error
	req := &protoutil.ListChannelsRequest{}
	req.Signer, err = o.getSigner()
	if err != nil {
		return nil, err
	}
	peers, err := o.getPeers()
	if err != nil {
		return nil, err
	}
	req.Peer = peers[0]
	return req, nil
}

func createChannel(channelID string, signer *protoutil.Signer, orderer organization, members []organization) error {
	req := &protoutil.CreateChannelRequest{
		ChannelId: channelID,
		ConsortiumName: "SampleConsortium",
		Signer: signer,
	}
	var err error
	req.Orderer, err = orderer.getOrderer(0)
	if err != nil {
		return err
	}
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

func joinChannel(channelID string, ordererOrg, org organization) error {
	signer, err := org.getSigner()
	if err != nil {
		return err
	}

	peers, err := org.getPeers()
	if err != nil {
		return err
	}

	orderer, err := ordererOrg.getOrderer(0)
	if err != nil {
		return err
	}
	req := &protoutil.JoinChannelRequest{
		ChannelId: channelID,
		Signer: signer,
		Peers: peers,
		Orderer: orderer,
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
		time.Sleep(500*time.Millisecond)
		fmt.Println("create channel", cm.ID, "success")
	}
}

func TestChannel_Join(t *testing.T) {
	joinChs := []struct{
		ChannelID  string
		Org  organization
	} {
		{"channel01", org1},
		{"channel01", org2},
		{"channel02", org1},
		{"channel02", org2},
		{"channel03", org1},
		{"channel03", org2},
	}
	for _, join := range joinChs {
		err := joinChannel(join.ChannelID, ordererOrg, join.Org)
		if err != nil {
			t.Fatal("Fail to join channel:", join.ChannelID, "org:", join.Org.Name, "mspid:", join.Org.MspID, "message:", err)
		}
		fmt.Println("success to join", join.ChannelID, "by", join.Org.Name)
		time.Sleep(500*time.Millisecond)
	}
}

func TestChannel_List(t *testing.T) {
	orgs := []organization{org1, org2}
	for _, org := range orgs {
		req, err := org.getListChannelRequest()
		if err != nil {
			t.Fatal(err)
		}
		rsp, err := gateway.ChannelList(context.Background(), req)
		if err != nil {
			t.Fatal("fail to list channel: ", err)
		}
		if rsp.Status != gateway.RESPONSE_OK {
			t.Fatal("invalid response status:", rsp.Status, rsp.Message)
		}
	}
}

func listChannel(org organization, expected []string) error {
	req, err := org.getListChannelRequest()
	if err != nil {
		return err
	}

	rsp, err := gateway.ChannelList(context.Background(), req)
	if err != nil {
		return err
	}

	if rsp.Status != gateway.RESPONSE_OK {
		return fmt.Errorf("invalid response status: %d, %s", rsp.Status, rsp.Message)
	}
	return nil
}

func TestChannel_Operation(t *testing.T) {
	testChannels := []struct{
		ChannelID  string
		Orgs   []organization
	}{
		{"channel01", []organization{org1, org2}},
		{"channel02", []organization{org2, org1}},
		{"channel03", []organization{org1}},
		{"channel04", []organization{org2}},
	}

	getName:= func(orgs []organization) string {
		name := ""
		for _, org := range orgs {
			name = name + org.Name + ","
		}
		return name
	}
	// test create
	for _, channel := range testChannels {
		signer, err := channel.Orgs[0].getSigner()
		if err != nil {
			t.Fatal(err)
		}
		err = createChannel(channel.ChannelID, signer, ordererOrg, channel.Orgs)
		if err != nil {
			t.Fatal("create channel fail:", err)
		}
		fmt.Println("create channel success", "channelid:", channel.ChannelID, "orgs:", getName(channel.Orgs))
		time.Sleep(500*time.Millisecond)
	}

	// test join
	for _, channel := range testChannels {
		for _, org := range channel.Orgs {
			err := joinChannel(channel.ChannelID, ordererOrg, org)
			if err != nil {
				t.Fatal("fail to join channel:", channel.ChannelID, "org:", org.Name)
			}
			fmt.Println("success to join channel:", channel.ChannelID, "org:", org.Name)
			time.Sleep(300*time.Millisecond)
		}
	}

	channelCheck := []struct{
		org  organization
		expected []string
	} {
		{org1, []string{"channel01", "channel02", "channel03"}},
		{org2, []string{"channel01", "channel02", "channel04"}},
	}
	// test list
	for _, check := range channelCheck {
		err := listChannel(check.org, check.expected)
		if err != nil {
			t.Fatal(err)
		}
	}

}
