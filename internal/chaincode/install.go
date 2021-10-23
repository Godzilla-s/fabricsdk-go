package chaincode

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/client/peer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	lb "github.com/hyperledger/fabric-protos-go/peer/lifecycle"
)

func Install(signer cryptoutil.Signer, committers []peer.Client, chaincode ChaincodeInstaller) (*protoutil.ChaincodeInstallResponse, error) {
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

	signedProp, err := cryptoutil.GetSignedProposal(prop, signer)
	if err != nil {
		return nil, err
	}

	var resp protoutil.ChaincodeInstallResponse
	var successCount int
	for _, committer := range committers {
		endorseCli, err := committer.GetEndorser()
		if err != nil {
			return nil, err
		}

		rsp, err := endorseCli.ProcessProposal(context.Background(), signedProp)
		if err != nil {
			return nil, err
		}

		result := &lb.InstallChaincodeResult{}
		err = proto.Unmarshal(rsp.Response.Payload, result)
		if err != nil {
			// TODO
		}

		if resp.Label == "" {
			resp.Label = result.Label
		}

		if resp.PackageId == "" {
			resp.PackageId = result.PackageId
		}
		if rsp.Response.Status == 200 {
			successCount++
		}
		resp.Results = append(resp.Results, &protoutil.ChaincodeInstallResponse_Result{
			Id: committer.GetAddress(),
			Status: rsp.Response.Status,
			Message: rsp.Response.Message,
		})
	}
	resp.Status = 200
	if successCount == 0 {
		resp.Status = 500
	} else if successCount != len(committers) {
		resp.Status = 204
	}
	return &resp, nil
}

func install(endorsers []pb.EndorserClient, signedProposal *pb.SignedProposal) (*pb.ProposalResponse, error) {

	return nil, nil
}