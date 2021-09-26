package chaincode

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/internal/client/peer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	lb "github.com/hyperledger/fabric-protos-go/peer/lifecycle"
)

// ChaincodeInstallResult
type ChaincodeInstallResult struct {
	Responses []pb.Response
	Lebel     string
	PackageID string
}

func Install(signer cryptoutil.Signer, commiters []peer.Client, chaincode ChaincodeInstaller) (*ChaincodeInstallResult, error) {
	installChaincodeArgs, err := chaincode.GetInstalledChaincode()
	if err != nil {
		return nil, err
	}
	installChaincodeArgsBytes, err := proto.Marshal(installChaincodeArgs)
	if err != nil {
		return nil, err
	}

	input := &pb.ChaincodeInput{Args: [][]byte{[]byte(LSCC_InstallChaincode), installChaincodeArgsBytes}}
	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       input,
		},
	}
	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}
	prop, _, err := utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, "", cis, creator)
	if err != nil {
		return nil, err
	}


	response := &ChaincodeInstallResult{
		Responses: make([]pb.Response, len(commiters)),
	}

	for i, committer := range commiters {
		endorseCli, err := committer.GetEndorser()
		if err != nil {
			response.Responses[i] = pb.Response{Status: 500, Message: err.Error()}
			continue
		}
		signedProp, err := cryptoutil.GetSignedProposal(prop, signer)
		if err != nil {
			return nil, err
		}
		rsp, err := endorseCli.ProcessProposal(context.Background(), signedProp)
		if err != nil {
			// fmt.Println("process error: ", err)
			response.Responses[i] = pb.Response{Status: 500, Message: err.Error()}
		} else {
			response.Responses[i] = *rsp.Response
			result := &lb.InstallChaincodeResult{}
			// fmt.Println(rsp.Response.Status, rsp.Response.Message)
			err = proto.Unmarshal(rsp.Response.Payload, result)
			if err == nil {
				response.PackageID = result.PackageId
				response.Lebel = result.Label
			}
		}
	}

	return response, nil
}
