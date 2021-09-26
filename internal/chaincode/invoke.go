package chaincode

import (
	"context"
	"encoding/json"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"strings"
	"sync"
	"time"
)

// ChaincodeSpec
type ChaincodeSpec struct {
	Name            string
	Version         string
	Lang            string
	Args            [][]byte
	Path            string
	Transient       string
	Collection      []byte
	PolicyMarshaled []byte
	IsInit          bool
	Timeout        time.Duration
}

func createInvocationProposal(signer cryptoutil.Signer, spec ChaincodeSpec, channelID string) (*pb.Proposal, string, error) {
	chaincodeLang := strings.ToUpper(spec.Lang)
	chaincodeSpec := &pb.ChaincodeSpec{
		Input: &pb.ChaincodeInput{
			Args:   spec.Args,
			IsInit: spec.IsInit,
		},
		Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value[chaincodeLang]),
		ChaincodeId: &pb.ChaincodeID{Path: spec.Path, Name: spec.Name, Version: spec.Version},
	}
	invocation := &pb.ChaincodeInvocationSpec{ChaincodeSpec: chaincodeSpec}

	// extract the transient field if it exists
	var tMap map[string][]byte
	if spec.Transient != "" {
		if err := json.Unmarshal([]byte(spec.Transient), &tMap); err != nil {
			return nil, "", errors.Wrap(err, "error parsing transient string")
		}
	}
	creator, err := signer.Serialize()
	if err != nil {
		return nil, "", errors.WithMessage(err, "fail to serialize")
	}
	return utils.CreateChaincodeProposalWithTxIDAndTransient(cb.HeaderType_ENDORSER_TRANSACTION, channelID, invocation, creator, "", tMap)
}

func invokeOrQeury(signer cryptoutil.Signer, cf *CommonFactory, spec ChaincodeSpec, channelID string, invoke bool) (*Response, error) {
	propsal, txID, err := createInvocationProposal(signer, spec, channelID)
	if err != nil {
		return nil, err
	}

	signeProp, err := cryptoutil.GetSignedProposal(propsal, signer)
	if err != nil {
		return nil, err
	}

	proposalResps, err := processProposal(signeProp, cf)
	if err != nil {
		return nil, err
	}

	if len(proposalResps) == 0 {
		// this should only happen if some new code has introduced a bug
		return nil, errors.New("no proposal responses received - this might indicate a bug")
	}
	proposalResp := proposalResps[0]
	response := &Response{TxID: txID, Response: proposalResp.Response}
	if proposalResp.Response.Status >= shim.ERRORTHRESHOLD {
		return response, nil
	}

	if invoke {
		err = broadcastProposalEnvelope(signer, propsal, cf, channelID, txID, spec.Timeout, proposalResps...)
		if err != nil {
			return response, err
		}
	}
	return response, nil
}

func processProposal(signedProp *pb.SignedProposal, cf *CommonFactory) ([]*pb.ProposalResponse, error) {
	responsesCh := make(chan *pb.ProposalResponse, len(cf.Endorsers))
	errorCh := make(chan error, len(cf.Endorsers))

	wg := sync.WaitGroup{}
	for _, endorser := range cf.Endorsers {
		wg.Add(1)
		go func(endorser pb.EndorserClient) {
			defer wg.Done()
			proposalResp, err := endorser.ProcessProposal(context.Background(), signedProp)
			if err != nil {
				errorCh <- err
				return
			}
			responsesCh <- proposalResp
		}(endorser)
	}
	wg.Wait()
	close(responsesCh)
	close(errorCh)

	for err := range errorCh {
		return nil, err
	}

	var responses []*pb.ProposalResponse
	for response := range responsesCh {
		responses = append(responses, response)
	}
	return responses, nil
}

func Invoke(signer cryptoutil.Signer, cf *CommonFactory, spec ChaincodeSpec, channelID string) (*Response, error) {
	resp, err := invokeOrQeury(signer, cf, spec, channelID, true)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func Query(signer cryptoutil.Signer, cf *CommonFactory, spec ChaincodeSpec, channelID string) (*Response, error) {
	return invokeOrQeury(signer, cf, spec, channelID, false)
}

type ProcessProposalResult struct {
	txID      string
	proposal   *pb.Proposal
	Responses []*pb.ProposalResponse
}

func SendTransaction(signer cryptoutil.Signer, cf *CommonFactory, spec ChaincodeSpec, channelID string) (*ProcessProposalResult, error) {
	proposal, txID, err := createInvocationProposal(signer, spec, channelID)
	if err != nil {
		return nil, err
	}

	signeProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, err
	}

	proposalResps, err := processProposal(signeProp, cf)
	if err != nil {
		return nil, err
	}

	result := &ProcessProposalResult{proposal: proposal, Responses: proposalResps, txID: txID}
	return result, nil
}

