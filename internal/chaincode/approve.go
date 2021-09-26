package chaincode

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/internal/chaincode/policy"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	lb "github.com/hyperledger/fabric-protos-go/peer/lifecycle"
	"github.com/pkg/errors"
)

func createPolicyBytes(signaturePolicy, channelConfigPolicy string) ([]byte, error) {
	if signaturePolicy == "" && channelConfigPolicy == "" {
		// no policy, no problem
		return nil, nil
	}

	if signaturePolicy != "" && channelConfigPolicy != "" {
		// mo policies, mo problems
		return nil, errors.New("cannot specify both \"--signature-policy\" and \"--channel-config-policy\"")
	}

	var applicationPolicy *pb.ApplicationPolicy
	if signaturePolicy != "" {
		signaturePolicyEnvelope, err := policy.FromString(signaturePolicy)
		if err != nil {
			return nil, errors.Errorf("invalid signature policy: %s", signaturePolicy)
		}

		applicationPolicy = &pb.ApplicationPolicy{
			Type: &pb.ApplicationPolicy_SignaturePolicy{
				SignaturePolicy: signaturePolicyEnvelope,
			},
		}
	}

	if channelConfigPolicy != "" {
		applicationPolicy = &pb.ApplicationPolicy{
			Type: &pb.ApplicationPolicy_ChannelConfigPolicyReference{
				ChannelConfigPolicyReference: channelConfigPolicy,
			},
		}
	}

	policyBytes := utils.MarshalOrPanic(applicationPolicy)
	return policyBytes, nil
}


func createApproveChaincodeProposal(signer cryptoutil.Signer, req *ApproveChaincodeRequest, channelID string) (*pb.Proposal, string, error) {
	var ccsrc *lb.ChaincodeSource
	if req.PackageID != "" {
		ccsrc = &lb.ChaincodeSource{
			Type: &lb.ChaincodeSource_LocalPackage{
				LocalPackage: &lb.ChaincodeSource_Local{
					PackageId: req.PackageID,
				},
			},
		}
	} else {
		ccsrc = &lb.ChaincodeSource{
			Type: &lb.ChaincodeSource_Unavailable_{
				Unavailable: &lb.ChaincodeSource_Unavailable{},
			},
		}
	}
	policyBytes, err := createPolicyBytes(req.SignPolicy, req.ChannelConfigPolicy)
	if err != nil {
		return nil, "", err
	}
	var ccpkg *pb.CollectionConfigPackage
	if len(req.CollectionConfig) > 0 {
		ccpkg, _, err = getCollectionConfigFromBytes(req.CollectionConfig)
	}
	args := &lb.ApproveChaincodeDefinitionForMyOrgArgs{
		Name:                req.Name,
		Version:             req.Version,
		Sequence:            req.Sequence,
		EndorsementPlugin:   req.EndorserPlugin,
		ValidationPlugin:    req.ValidationPlugin,
		ValidationParameter: policyBytes,
		InitRequired:        req.InitRequired,
		Collections: ccpkg,
		Source: ccsrc,
	}
	if req.WaitForEventTimeout > 0 {
		req.WaitForEvent = true
	}
	argsBytes, err := proto.Marshal(args)
	if err != nil {
		return nil, "", err
	}

	ccInput := &pb.ChaincodeInput{Args: [][]byte{[]byte(LSCC_ApproveFuncName), argsBytes}}

	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       ccInput,
		},
	}

	creator, err := signer.Serialize()
	if err != nil {
		return nil, "", err
	}
	return utils.CreateChaincodeProposalWithTxIDAndTransient(cb.HeaderType_ENDORSER_TRANSACTION, channelID, cis, creator, req.TxID, nil)
}


func Approve(signer cryptoutil.Signer, cf *CommonFactory, req *ApproveChaincodeRequest, channelID string) (*Response, error) {
	proposal, txID, err := createApproveChaincodeProposal(signer, req, channelID)
	if err != nil {
		return nil, err
	}
	signedProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, err
	}

	proposalResp, err := cf.Committer.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, errors.WithMessagef(err, "fail to process proposal")
	}

	if proposalResp.Response == nil {
		return nil, errors.Errorf("received proposal response with nil response")
	}

	if proposalResp.Response.Status != int32(cb.Status_SUCCESS) {
		return nil, errors.Errorf("proposal failed with status: %d - %s", proposalResp.Response.Status, proposalResp.Response.Message)
	}

	response := &Response{TxID: txID, Response: proposalResp.Response}
	// TODO: 处理发送至所有peer节点
	err = broadcastProposalEnvelope(signer, proposal, cf, channelID, txID, req.WaitForEventTimeout, proposalResp)
	if err != nil {
		return response, err
	}

	return response, nil
}
