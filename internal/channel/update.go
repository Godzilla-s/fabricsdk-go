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

// Create 创建通道
func Create(signer cryptoutil.Signer, ch ChannelEnvelope, client orderer.Client) error {
	chEnv, err := ch.CreateEnvelope()
	if err != nil {
		return err
	}

	signedEnv, err := sanityCheckAndSignConfigTx(chEnv, signer, ch.ChannelID())
	if err != nil {
		return  err
	}

	broadcastClient, err := client.GetBroadcastClient()
	if err != nil {
		return err
	}
	err = broadcastClient.Send(signedEnv)
	if err != nil {
		broadcastClient.Close()
		return err
	}
	broadcastClient.Close()
	return nil
}

func sanityCheckAndSignConfigTx(envConfigUpdate *cb.Envelope, signer cryptoutil.Signer, channelID string) (*cb.Envelope, error) {
	payload, err := utils.UnmarshalPayload(envConfigUpdate.Payload)
	if err != nil {
		return nil, errors.WithMessage(err, "bad payload")
	}

	if payload.Header == nil || payload.Header.ChannelHeader == nil {
		return nil,  errors.WithMessage(err, "bad header")
	}

	ch, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, errors.WithMessage(err,"could not unmarshall channel header")
	}

	if ch.Type != int32(cb.HeaderType_CONFIG_UPDATE) {
		return nil, errors.WithMessage(err,"bad type")
	}

	if ch.ChannelId == "" {
		return nil, errors.WithMessage(err,"empty channel id")
	}

	// Specifying the chainID on the CLI is usually redundant, as a hack, set it
	// here if it has not been set explicitly
	if channelID == "" {
		channelID = ch.ChannelId
	}

	if ch.ChannelId != channelID {
		return nil, errors.New(fmt.Sprintf("mismatched channel Name %s != %s", ch.ChannelId, channelID))
	}

	configUpdateEnv, err := utils.UnmarshalConfigUpdateEnvelope(payload.Data)
	if err != nil {
		return nil, errors.WithMessage(err,"Bad config update env")
	}

	sigHeader, err := utils.NewSignatureHeader(signer)
	if err != nil {
		return nil, err
	}

	configSig := &cb.ConfigSignature{
		SignatureHeader: utils.MarshalOrPanic(sigHeader),
	}

	configSig.Signature, err = signer.Sign(utils.ConcatenateBytes(configSig.SignatureHeader, configUpdateEnv.ConfigUpdate))
	if err != nil {
		return nil, err
	}

	configUpdateEnv.Signatures = append(configUpdateEnv.Signatures, configSig)

	return utils.CreateSignedEnvelope(cb.HeaderType_CONFIG_UPDATE, channelID, signer, configUpdateEnv, 0, 0)
}

// Update 更新通道
func Update(signer cryptoutil.Signer, ch ChannelEnvelope, oClient orderer.Client) error {
	chEnv, err := ch.CreateEnvelope()
	if err != nil {
		return errors.WithMessage(err, "create envelope")
	}

	signedEnv, err := sanityCheckAndSignConfigTx(chEnv, signer, ch.ChannelID())
	if err != nil {
		return errors.WithMessage(err, "sanity check and sign configtx")
	}

	broadcast, err := oClient.GetBroadcastClient()
	if err != nil {
		return err
	}
	defer broadcast.Close()

	err = broadcast.Send(signedEnv)
	if err != nil {
		return err
	}
	return nil
}

func Join(signer cryptoutil.Signer, pClients []peer.Client, oClient orderer.Client, channelID string) ([]*pb.Response, error) {
	ds, err := oClient.GetDeliverClient(signer, channelID, true)
	if err != nil {
		return nil, err
	}

	//fmt.Println("==> channel id:", channelID)
	block, err := ds.GetSpecifiedBlock(0)
	if err != nil {
		return nil, fmt.Errorf("fail to get config block from channel %s with error: %v", channelID, err)
	}
	//fmt.Println("==> block number: ", block.Header.Number)

	blockBytes, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}

	var responses []*pb.Response
	for _, peerClient := range pClients {
		endorser, err := peerClient.GetEndorser()
		if err != nil {
			responses = append(responses, &pb.Response{Status: 500, Message: err.Error(), Payload: []byte(peerClient.GetAddress())})
			continue
		}
		signedProp, err := createJoinChannelProposal(signer, blockBytes)
		if err != nil {
			responses = append(responses, &pb.Response{Status: 500, Message: err.Error(), Payload: []byte(peerClient.GetAddress()) })
			continue
		}
		r, err := endorser.ProcessProposal(context.Background(), signedProp)
		if err != nil {
			responses = append(responses, &pb.Response{Status: 500, Message: fmt.Sprintf("process proposal, error: %v", err), Payload: []byte(peerClient.GetAddress())})
			continue
		}
		responses = append(responses, r.Response)
	}
	return responses, nil
}

func createJoinChannelProposal(signer cryptoutil.Signer, block []byte) (*pb.SignedProposal, error) {
	input := &pb.ChaincodeInput{Args: [][]byte{[]byte(CSCC_JoinChannel), block}}

	spec := &pb.ChaincodeSpec{
		Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value["GOLANG"]),
		ChaincodeId: &pb.ChaincodeID{Name: "cscc"},
		Input:       input,
	}
	invocation := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}
	creator, err := signer.Serialize()
	if err != nil {
		return nil, err
	}
	prop, _, err := utils.CreateProposalFromCIS(cb.HeaderType_CONFIG, "", invocation, creator)
	if err != nil {
		return nil, fmt.Errorf("error creating proposal for join %s", err)
	}

	signedProp, err := cryptoutil.GetSignedProposal(prop, signer)
	if err != nil {
		return nil, fmt.Errorf("error creating signed proposal %s", err)
	}
	return signedProp, nil
}

