package example

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/gateway"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"testing"
)

func installChaincode(t *testing.T, org organization) {
	signer, err := org.getSigner()
	if err != nil {
		t.Fatal("fail to get signer: err", err, "org:", org.Name)
	}

	peers, err := org.getPeers()
	if err != nil {
		t.Fatal(err)
	}
	req := &protoutil.ChaincodeInstallRequest{
		Signer: signer,
		Peers: peers,
	}
	resp, err := gateway.ChaincodeInstall(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != gateway.RESPONSE_OK {
		t.Fatal("fail to install all", "status", resp.Status, "results:", resp.Results)
	}
}

func approveChaincode(t *testing.T, org organization, channelID string, ) {
	signer, err := org.getSigner()
	if err != nil {
		t.Fatal(err)
	}

	peers, err := org.getPeers()
	if err != nil {
		t.Fatal(err)
	}
	req := &protoutil.ChaincodeApproveRequest{
		Signer: signer,
		ChannelId: channelID,
		Committer: peers[0],
		Definition: &protoutil.DefinitionArgs{},
		PackageId: "",
	}

	rsp, err := gateway.ChaincodeApprove(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if rsp.Status != gateway.RESPONSE_OK {
		t.Fatal("approve chaincode fail, status:", rsp.Status, "message:", rsp.Message)
	}
}

func commitChaincode(t *testing.T, endorserOrgs []organization, ordererOrg organization, channelID string) {
	signer, err := endorserOrgs[0].getSigner()
	if err != nil {
		t.Fatal(err)
	}

	req := &protoutil.ChaincodeCommitRequest{
		Signer: signer,
		ChannelId: channelID,
	}

	rsp, err := gateway.ChaincodeCommit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if rsp.Status != gateway.RESPONSE_OK {
		t.Fatal("fail to commit chaincode: status:", rsp.Status, "message:", rsp.Message)
	}
}

func TestChaincode_Operator(t *testing.T) {

}
