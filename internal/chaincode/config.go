package chaincode

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/client/delivegroup"
	"github.com/godzilla-s/fabricsdk-go/internal/client/orderer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"time"
)

const (
	LifeCycleName = "_lifecycle"
	LSCC_InstallChaincode             = "InstallChaincode"
	LSCC_ApproveFuncName              = "ApproveChaincodeDefinitionForMyOrg"
	LSCC_CommitFuncName               = "CommitChaincodeDefinition"
	LSCC_CheckCommitReadinessFuncName = "CheckCommitReadiness"
	LSCC_QueryInstalledChaincode      = "QueryInstalledChaincodes"
	LSCC_QueryApprivedChaincode       = "QueryApprovedChaincodeDefinition"
)

type CommonFactory struct {
	Committer     pb.EndorserClient
	Endorsers     []pb.EndorserClient
	Delivers      []pb.DeliverClient
	PeerAddresses []string
	OClient    		orderer.Client
	TLSCert       tls.Certificate
}

func (cf *CommonFactory) deliver(signer cryptoutil.Signer, channelID, txID string, waitTime time.Duration) error {
	var cancel context.CancelFunc
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()
	dg := delivegroup.NewDeliverGroup(cf.Delivers, cf.PeerAddresses, signer, cf.TLSCert, channelID, txID)
	err := dg.Connect(ctx)
	if err != nil {
		return errors.WithMessage(err, "fail to connect deliver group")
	}
	return nil
}


type Response struct {
	TxID     string
	Response *pb.Response
}

func createSignedTx(proposal *pb.Proposal, signer cryptoutil.Signer, resps ...*pb.ProposalResponse) (*common.Envelope, error) {
	if len(resps) == 0 {
		return nil, errors.New("at least one proposal response is required")
	}

	hdr, err := utils.UnmarshalHeader(proposal.Header)
	if err != nil {
		return nil, errors.Wrap(err, "get header")
	}

	payload, err := utils.UnmarshalChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "get chaincode proposal payload")
	}
	signedBytes, err := signer.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "get creator")
	}
	shdr, err := utils.UnmarshalSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, errors.Wrap(err, "get sign header")
	}
	if bytes.Compare(signedBytes, shdr.Creator) != 0 {
		return nil, errors.New("signer must be the same as the one referenced in the header")
	}

	// ensure that all actions are bitwise equal and that they are successful
	var a1 []byte
	for n, r := range resps {
		if r.Response.Status < 200 || r.Response.Status >= 400 {
			return nil, errors.Errorf("proposal response was not successful, error code %d, msg %s", r.Response.Status, r.Response.Message)
		}

		if n == 0 {
			a1 = r.Payload
			continue
		}

		if !bytes.Equal(a1, r.Payload) {
			fmt.Println(a1)
			fmt.Println(r.Payload)
			return nil, errors.New("ProposalResponsePayloads do not match")
		}
	}

	// fill endorsements
	endorsements := make([]*pb.Endorsement, len(resps))
	for n, r := range resps {
		endorsements[n] = r.Endorsement
	}

	// create ChaincodeEndorsedAction
	cea := &pb.ChaincodeEndorsedAction{ProposalResponsePayload: resps[0].Payload, Endorsements: endorsements}

	// obtain the bytes of the proposal payload that will go to the transaction
	propPayloadBytes, err := utils.GetBytesProposalPayloadForTx(payload)
	if err != nil {
		return nil, err
	}

	// serialize the chaincode action payload
	ccpayload := &pb.ChaincodeActionPayload{ChaincodeProposalPayload: propPayloadBytes, Action: cea}
	capBytes, err := utils.GetBytesChaincodeActionPayload(ccpayload)
	if err != nil {
		return nil, err
	}

	// create a transaction
	taa := &pb.TransactionAction{Header: hdr.SignatureHeader, Payload: capBytes}
	taas := make([]*pb.TransactionAction, 1)
	taas[0] = taa
	tx := &pb.Transaction{Actions: taas}

	// serialize the tx
	txBytes, err := utils.GetBytesTransaction(tx)
	if err != nil {
		return nil, err
	}

	// create the payload
	payl := &common.Payload{Header: hdr, Data: txBytes}
	paylBytes, err := utils.GetBytesPayload(payl)
	if err != nil {
		return nil, err
	}

	// sign the payload
	sig, err := signer.Sign(paylBytes)
	if err != nil {
		return nil, err
	}

	// here's the envelope
	return &common.Envelope{Payload: paylBytes, Signature: sig}, nil
}

// broadcastProposalEnvelope 将处理的交易包广播至orderer节点
func broadcastProposalEnvelope(signer cryptoutil.Signer, proposal *pb.Proposal, cf *CommonFactory, channelID, txID string, timeout time.Duration, responses ...*pb.ProposalResponse) error {
	env, err := createSignedTx(proposal, signer, responses...)
	if err != nil {
		return err
	}

	broadcastClient, err := cf.OClient.GetBroadcastClient()
	if err != nil {
		return err
	}

	var dg *delivegroup.DeliverGroup
	var ctx context.Context
	if timeout > 0 {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		dg = delivegroup.NewDeliverGroup(cf.Delivers, cf.PeerAddresses, signer, cf.TLSCert, channelID, txID)
		err := dg.Connect(ctx)
		if err != nil {
			return errors.WithMessage(err, "fail to connect deliver group")
		}
	}

	err = broadcastClient.Send(env)
	if err != nil {
		broadcastClient.Close()
		return err
	}
	broadcastClient.Close()
	if dg != nil && timeout > 0 {
		err = dg.Wait(ctx)
		if err != nil {
			return errors.WithMessage(err, "fail to wait for delivering")
		}
	}
	return nil
}
