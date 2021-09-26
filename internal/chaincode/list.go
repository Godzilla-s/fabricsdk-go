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

func createQueryInstalledProposal(signer cryptoutil.Signer) (*pb.Proposal, error) {
	args := &lb.QueryInstalledChaincodesArgs{}
	argsBytes, err := proto.Marshal(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal args")
	}

	ccInput := &pb.ChaincodeInput{
		Args: [][]byte{[]byte(LSCC_QueryInstalledChaincode), argsBytes},
	}

	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       ccInput,
		},
	}

	creator, err := signer.Serialize()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to serialize identity")
	}

	prop, _, err := utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, "", cis, creator)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create ChaincodeInvocationSpec proposal")
	}
	return prop, nil
}

type InstalledChaincodeList struct {
	lb.QueryInstalledChaincodesResult
}

func QueryInstalled(signer cryptoutil.Signer, endorseCli pb.EndorserClient) (*InstalledChaincodeList, error) {
	proposal, err := createQueryInstalledProposal(signer)
	if err != nil {
		return nil, err
	}
	signedProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create signed proposal")
	}

	resp, err := endorseCli.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to endorse proposal")
	}

	if resp.Response == nil {
		return nil, errors.New("received proposal response with nil response")
	}

	if resp.Response.Status != int32(cb.Status_SUCCESS) {
		return nil, errors.Errorf("query failed with status: %d - %s", resp.Response.Status, resp.Response.Message)
	}

	qicr := &lb.QueryInstalledChaincodesResult{}
	err = proto.Unmarshal(resp.Response.Payload, qicr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal proposal response's response payload")
	}

	response := &InstalledChaincodeList{}
	response.InstalledChaincodes = qicr.InstalledChaincodes
	return response, nil
}

func createQueryApprovedProposal(signer cryptoutil.Signer, name, channelID string) (*pb.Proposal, error) {
	args := &lb.QueryApprovedChaincodeDefinitionArgs{
		Name:     name,
		Sequence: 0,
	}
	argsBytes, err := proto.Marshal(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal args")
	}
	ccInput := &pb.ChaincodeInput{Args: [][]byte{[]byte(LSCC_QueryApprivedChaincode), argsBytes}}

	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       ccInput,
		},
	}

	creator, err := signer.Serialize()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to serialize identity")
	}
	proposal, _, err := utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, channelID, cis, creator)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create ChaincodeInvocationSpec proposal")
	}
	return proposal, nil
}

type ApprovedChaincodeList struct {
	*lb.QueryApprovedChaincodeDefinitionResult
}

func QueryApproved(signer cryptoutil.Signer, endorser pb.EndorserClient, name, channelID string) (*ApprovedChaincodeList, error) {
	proposal, err := createQueryApprovedProposal(signer, name, channelID)
	if err != nil {
		return nil, err
	}
	signedProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, err
	}
	proposalResp, err := endorser.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, err
	}
	if proposalResp.Response == nil {
		return nil, errors.New("received proposal response with nil response")
	}

	if proposalResp.Response.Status != int32(cb.Status_SUCCESS) {
		return nil, errors.Errorf("query failed with status: %d - %s", proposalResp.Response.Status, proposalResp.Response.Message)
	}

	result := &lb.QueryApprovedChaincodeDefinitionResult{}
	err = proto.Unmarshal(proposalResp.Response.Payload, result)
	if err != nil {
		return nil, err
	}
	return &ApprovedChaincodeList{result}, nil
}

func createQueryCommittedChaincodeProposal(signer cryptoutil.Signer, name, channelID string) (*pb.Proposal, error) {
	var function string
	var args proto.Message

	if name != "" {
		function = "QueryChaincodeDefinition"
		args = &lb.QueryChaincodeDefinitionArgs{
			Name: name,
		}
	} else {
		function = "QueryChaincodeDefinitions"
		args = &lb.QueryChaincodeDefinitionsArgs{}
	}

	argsBytes, err := proto.Marshal(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal args")
	}
	ccInput := &pb.ChaincodeInput{Args: [][]byte{[]byte(function), argsBytes}}

	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       ccInput,
		},
	}
	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}
	proposal, _, err := utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, channelID, cis, creator)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create ChaincodeInvocationSpec proposal")
	}
	return proposal, nil
}

type CommittedChaincodeList struct {
	lb.QueryChaincodeDefinitionsResult
	lb.QueryChaincodeDefinitionResult
}

func QueryCommitted(signer cryptoutil.Signer, endorseCli pb.EndorserClient, channelID string, opts ...Option) (*CommittedChaincodeList, error) {
	req := &QueryCommittedChaincodeRequest{}
	for _, opt := range opts {
		req = opt(req).(*QueryCommittedChaincodeRequest)
	}

	proposal, err := createQueryCommittedChaincodeProposal(signer, req.Name, channelID)
	if err != nil {
		return nil, err
	}

	signedProposal, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create signed proposal")
	}

	resp, err := endorseCli.ProcessProposal(context.Background(), signedProposal)
	if err != nil {
		return nil, err
	}
	if resp.Response == nil {
		return nil, errors.New("received proposal response with nil response")
	}

	if resp.Response.Status != int32(cb.Status_SUCCESS) {
		return nil, errors.Errorf("query failed with status: %d - %s", resp.Response.Status, resp.Response.Message)
	}

	var list CommittedChaincodeList
	if req.Name == "" {
		result := &lb.QueryChaincodeDefinitionsResult{}
		err := proto.Unmarshal(resp.Response.Payload, result)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal proposal response's response payload")
		}
		list.ChaincodeDefinitions = result.ChaincodeDefinitions
	}
	result := &lb.QueryChaincodeDefinitionResult{}
	err = proto.Unmarshal(resp.Response.Payload, result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal proposal response's response payload")
	}
	return &list, nil
}

func createCommitReadinessProposal(signer cryptoutil.Signer, cr *CheckCommitReadinessRequest, channelID string) (*pb.Proposal, error) {
	args := &lb.CheckCommitReadinessArgs{
		Name:              cr.Name,
		Version:           cr.Version,
		Sequence:          cr.Sequence,
		InitRequired:      cr.InitRequired,
		EndorsementPlugin: cr.EndorsementPlugin,
	}
	argsBytes, err := proto.Marshal(args)
	if err != nil {
		return nil, err
	}
	ccInput := &pb.ChaincodeInput{Args: [][]byte{[]byte(LSCC_CheckCommitReadinessFuncName), argsBytes}}

	cis := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{Name: LifeCycleName},
			Input:       ccInput,
		},
	}

	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	proposal, _, err := utils.CreateChaincodeProposalWithTxIDAndTransient(cb.HeaderType_ENDORSER_TRANSACTION, channelID, cis, creator, "", nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create ChaincodeInvocationSpec proposal")
	}

	return proposal, nil
}

type CheckCommitReadinessResult struct {
	lb.CheckCommitReadinessResult
}

func CheckCommitReadiness(signer cryptoutil.Signer, endorseCli pb.EndorserClient, channelID string, opts ...Option) (*CheckCommitReadinessResult, error) {
	req := &CheckCommitReadinessRequest{}
	for _, opt := range opts {
		req = opt(req).(*CheckCommitReadinessRequest)
	}
	if req.Sequence == 0 {
		req.Sequence = 1
	}
	proposal, err := createCommitReadinessProposal(signer, req, channelID)
	if err != nil {
		return nil, err
	}

	signeProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create signed proposal")
	}

	proposalResponse, err := endorseCli.ProcessProposal(context.Background(), signeProp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to endorse proposal")
	}

	if proposalResponse.Response == nil {
		return nil, errors.New("received proposal response with nil response")
	}

	if proposalResponse.Response.Status != int32(cb.Status_SUCCESS) {
		return nil, errors.Errorf("query failed with status: %d - %s", proposalResponse.Response.Status, proposalResponse.Response.Message)
	}

	result := &lb.CheckCommitReadinessResult{}
	err = proto.Unmarshal(proposalResponse.Response.Payload, result)
	if err != nil {
		return nil, err
	}

	return &CheckCommitReadinessResult{*result}, nil
}
