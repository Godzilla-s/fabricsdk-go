package example

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/gateway"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"testing"
)

var (
	CLUSTER_NODE = ""
)

var (
	org1 = protoutil.Organization{
		Name: "org1",
		MspId: "Org1MSP",
		RootCert: []byte(""),
		TlsRootCert: []byte(""),
	}

	peer0org1 = protoutil.Peer{
		Url: CLUSTER_NODE,
		HostName: "peer0.org1.example.com",
		TlsRootCert: []byte(""),
	}

	orderer0 = protoutil.Orderer{
		Url: CLUSTER_NODE,
		HostName: "orderer0.example.com",
		TlsRootCert: []byte(""),
	}
)

func TestChannel_Create(t *testing.T) {
	req := &protoutil.CreateChannelRequest{
		Orderer: &orderer0,
		ChannelId: "mychannel",
		ConsortiumName: "SampleConsortium",
		Members: []*protoutil.Organization{&org1},
	}

	resp, err := gateway.ChannelCreate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	_ = resp
}