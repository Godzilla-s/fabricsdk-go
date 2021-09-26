package gateway

import (
	"bytes"
	"context"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/channel"
	"github.com/godzilla-s/fabricsdk-go/internal/client/orderer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/golang/protobuf/proto"
)

// ProposalInitiate 初始提案，如组织加入通道，加入联盟，组织被通道或者联盟删除
func ProposalInitiate(ctx context.Context, req *protoutil.ProposalInitRequest) (*protoutil.ProposalEnvelope, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}

	ordererClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, err
	}
	lastConfigBlock, err := channel.FetchConfig(signer, ordererClient, req.ChannelId)
	if err != nil {
		return nil, err
	}

	var updateEnvelope *channel.UpdateEnvelope
	switch req.Proposal.Type {
	case protoutil.ProposalType_Channel_AddPeerOrg:
		newOrg := req.Proposal.GetNewOrg()
		if newOrg == nil {
			return nil, fmt.Errorf("missing proposal content when type is %v", req.Proposal.Type.String())
		}
		updateEnvelope, err = channel.ChannelAddOrg(lastConfigBlock, createChannelOrg(newOrg), req.ChannelId)
	case protoutil.ProposalType_Channel_RemovePeerOrg:
		removedOrgName := req.Proposal.GetRemovedOrgName()
		if removedOrgName == "" {
			return nil, fmt.Errorf("missing proposal content when type is %v", req.Proposal.Type.String())
		}
		updateEnvelope, err = channel.ChannelRemoveOrg(lastConfigBlock, removedOrgName, req.ChannelId)
	case protoutil.ProposalType_Consortium_AddPeerOrg:
		newOrg := req.Proposal.GetNewOrg()
		if newOrg == nil {
			return nil, fmt.Errorf("missing proposal content when type is %v", req.Proposal.Type.String())
		}
		updateEnvelope, err = channel.ConsortiumAddOrg(lastConfigBlock, createChannelOrg(newOrg), req.ConsortiumName, req.ChannelId)
	case protoutil.ProposalType_Consortium_RemovePeerOrg:
		removedOrgName := req.Proposal.GetRemovedOrgName()
		if removedOrgName == "" {
			return nil, fmt.Errorf("missing proposal content when type is %v", req.Proposal.Type.String())
		}
		updateEnvelope, err = channel.ConsortiumRemoveOrg(lastConfigBlock, removedOrgName, req.ConsortiumName, req.ChannelId)
	}
	if err != nil {
		return nil, err
	}
	proposal := updateEnvelope.GetUpdates()
	sig, err := channel.SignUpdateConfig(signer, proposal)
	if err != nil {
		return nil, err
	}
	sigBytes, err := proto.Marshal(sig)
	if err != nil {
		return nil, err
	}
	proposalHash, err := cryptoutil.Hash(proposal, cryptoutil.SHA2_256)
	envelope := &protoutil.ProposalEnvelope{
		ChannelId: req.ChannelId,
		Proposal: proposal,
		Sign: &protoutil.ProposalSignature{
			ProposalHash: proposalHash,
			Creator: signer.GetMSPId(),
			Signature: sigBytes,
		},
	}

	return envelope, nil
}


// ProposalSign 提案签名
func ProposalSign(ctx context.Context, req *protoutil.ProposalSignRequest) (*protoutil.ProposalSignature, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}

	sig, err := channel.SignUpdateConfig(signer, req.Envelope.Proposal)
	if err != nil {
		return nil, err
	}
	sigBytes, err := proto.Marshal(sig)
	if err != nil {
		return nil, err
	}
	proposalHash, _ := cryptoutil.Hash(req.Envelope.Proposal, cryptoutil.SHA2_256)
	proposalSig := &protoutil.ProposalSignature{
		ProposalHash: proposalHash,
		Creator: signer.GetMSPId(),
		Signature: sigBytes,
	}
	return proposalSig, nil
}

// ProposalSubmit 提案提交
func ProposalSubmit(ctx context.Context, req *protoutil.ProposalSubmitRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}

	ordererClient, err := orderer.New(req.Orderer.Url, req.Orderer.HostName, req.Orderer.TlsRootCert)
	if err != nil {
		return nil, err
	}
	sponsor := req.Envelope.Sign
	if sponsor.Creator != signer.GetMSPId() {
		return nil, fmt.Errorf("proposal submit must be the same with sponsor, submit: %s, sponsor:%s", signer.GetMSPId(), sponsor.Creator)
	}
	sigs := make(map[string][]byte)
	proposalHash := req.Envelope.Sign.ProposalHash
	sigs[req.Envelope.Sign.Creator] = req.Envelope.Sign.Signature
	for _, sig := range req.Sigs {
		if bytes.Equal(proposalHash, sig.ProposalHash) {
			sigs[sig.Creator] = sig.Signature
		}
	}

	updateEnvelope, err := channel.CreateUpdateEnvelope(req.Envelope.Proposal, sigs, req.Envelope.ChannelId)
	if err != nil {
		return nil, err
	}
	err = channel.Update(signer, updateEnvelope, ordererClient)
	if err != nil {
		return &protoutil.Response{Status: 500, Message: err.Error()}, nil
	}
	return &protoutil.Response{Status: 200}, nil
}