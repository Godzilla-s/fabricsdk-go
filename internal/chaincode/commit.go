package chaincode

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	lb "github.com/hyperledger/fabric-protos-go/peer/lifecycle"
	"github.com/pkg/errors"
)

func createCommitProposal(signer cryptoutil.Signer, cr *CommitChaincodeRequest, channelID string) (proposal *pb.Proposal, txID string, err error) {
	policyBytes, err := createPolicyBytes(cr.SignPolicy, cr.ChannelConfigPolicy)
	if err != nil {
		return nil, "", err
	}

	var ccpkg *pb.CollectionConfigPackage
	if len(cr.CollectionConfig) > 0 {
		ccpkg, _, err = getCollectionConfigFromBytes(cr.CollectionConfig)
		if err != nil {
			return nil, "", err
		}
	}
	args := &lb.CommitChaincodeDefinitionArgs{
		Name:                cr.Name,
		Version:             cr.Version,
		Sequence:            cr.Sequence,
		EndorsementPlugin:   cr.EndorsementPlugin,
		ValidationPlugin:    cr.ValidationPlugin,
		ValidationParameter: policyBytes,
		InitRequired:        cr.InitRequired,
		Collections:         ccpkg,
	}

	argsBytes, err := proto.Marshal(args)
	if err != nil {
		return nil, "", err
	}
	ccInput := &pb.ChaincodeInput{Args: [][]byte{[]byte(LSCC_CommitFuncName), argsBytes}}

	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       ccInput,
		},
	}

	creatorBytes, err := signer.Serialize()
	if err != nil {
		return nil, "", errors.WithMessage(err, "failed to serialize identity")
	}

	proposal, txID, err = utils.CreateChaincodeProposalWithTxIDAndTransient(cb.HeaderType_ENDORSER_TRANSACTION, channelID, cis, creatorBytes, cr.TxID, nil)
	if err != nil {
		return nil, "", errors.WithMessage(err, "failed to create ChaincodeInvocationSpec proposal")
	}
	return proposal, txID, nil
}

func Commit(signer cryptoutil.Signer, cf *CommonFactory, req *CommitChaincodeRequest, channelID string) (*Response, error) {
	proposal, txID,  err := createCommitProposal(signer, req, channelID)
	if err != nil {
		return nil, err
	}

	signedProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, err
	}

	var responses []*pb.ProposalResponse
	for _, commit := range cf.Endorsers {
		resp, err := commit.ProcessProposal(context.Background(), signedProp)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to endorse proposal")
		}

		if resp.Response.Status != int32(cb.Status_SUCCESS) {
			return nil, errors.Errorf("proposal failed with status: %d - %s", resp.Response.Status, resp.Response.Message)
		}
		responses = append(responses, resp)
	}

	err = broadcastProposalEnvelope(signer, proposal, cf, channelID, txID, req.WaitForEventTimeout, responses...)
	if err != nil {
		return nil, err
	}
	resp := &Response{TxID: txID, Response: responses[0].Response}
	return resp, nil
}

