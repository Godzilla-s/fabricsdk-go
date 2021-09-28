package channel

import (
	"context"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/client/orderer"
	"github.com/godzilla-s/fabricsdk-go/internal/client/peer"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

func createListChannelProposal(signer cryptoutil.Signer) (*pb.Proposal, error) {
	invocation := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]),
			ChaincodeId: &pb.ChaincodeID{Name: CSCC},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte(CSCC_GetChannels)}},
		},
	}
	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}
	prop, _,  err := utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, "", invocation, creator)
	if err != nil {
		return nil, err
	}
	return prop, nil
}

// List 列出节点加入的通道
func List(signer cryptoutil.Signer, pClient peer.Client) (*pb.ChannelQueryResponse, error) {
	proposal, err := createListChannelProposal(signer)
	if err != nil {
		return nil, err
	}

	signedProp, err := cryptoutil.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, err
	}
	endorser, err := pClient.GetEndorser()
	if err != nil {
		return nil, err
	}
	proposalResp, err := endorser.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, fmt.Errorf("Failed sending proposal, got %s", err)
	}

	if proposalResp.Response == nil || proposalResp.Response.Status != 200 {
		return nil, fmt.Errorf("Received bad response, status %d: %s", proposalResp.Response.Status, proposalResp.Response.Message)
	}
	var channelQueryResponse pb.ChannelQueryResponse
	err = proto.Unmarshal(proposalResp.Response.Payload, &channelQueryResponse)
	if err != nil {
		return nil, fmt.Errorf("Cannot read channels list response, %s", err)
	}

	return &channelQueryResponse, nil
}


func createGetChannelInfoProposal(signer cryptoutil.Signer, channelID string) (*pb.Proposal, error) {
	invocation := &pb.ChaincodeInvocationSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]),
			ChaincodeId: &pb.ChaincodeID{Name: "qscc"},
			Input:       &pb.ChaincodeInput{Args: [][]byte{[]byte(CSCC_GetChainInfo), []byte(channelID)}},
		},
	}

	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	prop, _, err := utils.CreateProposalFromCIS(cb.HeaderType_ENDORSER_TRANSACTION, "", invocation, creator)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot create proposal")
	}
	return prop, nil
}

type BlockChannelInfo struct {
	cb.BlockchainInfo
}

// GetInfo 获取通道信息
func GetInfo(signer cryptoutil.Signer, endorser pb.EndorserClient, channelID string) (*cb.BlockchainInfo, error) {
	proposal, err := createGetChannelInfoProposal(signer, channelID)
	if err != nil {
		return nil, err
	}

	signedProp, err := utils.GetSignedProposal(proposal, signer)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot create signed proposal")
	}

	resp, err := endorser.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return nil, err
	}

	if resp.Response == nil || resp.Response.Status != 200 {
		return nil, errors.Errorf("received bad response, status %d: %s", resp.Response.Status, resp.Response.Message)
	}
	binfo := cb.BlockchainInfo{}
	err = proto.Unmarshal(resp.Response.Payload, &binfo)
	if err != nil {
		return nil, err
	}
	return &binfo, nil
}

// FetchBlock 获取区块
func FetchBlock(signer cryptoutil.Signer, oClient orderer.Client, channelID string, blockNum uint64) (*cb.Block, error) {
	if oClient == nil {
		return nil, fmt.Errorf("nil orderer client")
	}

	deliverCli, err := oClient.GetDeliverClient(signer, channelID, true)
	if err != nil {
		return nil, err
	}

	block, err := deliverCli.GetSpecifiedBlock(blockNum)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// 获取通道创世区块（首区块）
func FetchConfig(signer cryptoutil.Signer, ordererCli orderer.Client, channelID string) (*cb.Block, error) {
	deliverCli, err := ordererCli.GetDeliverClient(signer, channelID, true)
	if err != nil {
		return nil, err
	}
	newestBlock, err := deliverCli.GetNewestBlock()
	if err != nil {
		return nil, err
	}
	lastConfigIdx, err := utils.GetLastConfigIndexFromBlock(newestBlock)
	if err != nil {
		return nil, err
	}

	return deliverCli.GetSpecifiedBlock(lastConfigIdx)
}