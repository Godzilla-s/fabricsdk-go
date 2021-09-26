package block

import (
	"github.com/godzilla-s/fabricsdk-go/internal/rwsetutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

type Block struct {
	Header  *cb.BlockHeader
	Metadata *cb.BlockMetadata
	Data     []*Envelope
}

type Envelope struct {
	Payload   Payload
	Signature []byte
}

func (env Envelope) GetTxType() string {
	switch env.Payload.Header.ChannelHeader.Type {
	case 2:
		return "CONFIG_UPDATE"
	case 3:
		return "ENDORSER_TRANSACTION"
	case 4:
		return "ORDERER_TRANSACTION"
	default:
		return "UNKNOWN TRANSACTION TYPE"
	}
}

type Payload struct {
	Header      *PayloadHeader
	Transaction Transaction
}

type PayloadHeader struct {
	ChannelHeader   *cb.ChannelHeader
	SignatureHeader SignatureHeader
}

type Transaction struct {
	Actions []*TransactionAction
}

type TransactionAction struct {
	Header  *SignatureHeader
	Payload ChaincodeActionPayload
}

type ChaincodeProposalPayload struct {
	TransientMap map[string][]byte
	Input        *pb.ChaincodeSpec
}

type ChaincodeActionPayload struct {
	Action                   *ChaincodeEndorserAction
	ChaincodeProposalPayload *ChaincodeProposalPayload
}

type ChaincodeEndorserAction struct {
	Endorsements            []Endorsement
	ProposalResponsePayload *ProposalResponsePayload
}

type ProposalResponsePayload struct {
	Extension    ChaincodeAction
	ProposalHash []byte
}

type ChaincodeAction struct {
	Response    *pb.Response
	ChaincodeID *pb.ChaincodeID
	Events      []byte
	Results     *rwsetutil.TxRwSet
}


type SignatureHeader struct {
	Creator *msp.SerializedIdentity
	Nonce   []byte
}

// Endorsement 背书
type Endorsement struct {
	Endorser  *msp.SerializedIdentity
	Signature []byte
}

func (b *Block) GetChannelID() string {
	return b.Data[0].Payload.Header.ChannelHeader.ChannelId
}

func UnmarshalEnvelope(env *cb.Envelope) (*Envelope, error) {
	envelope := &Envelope{Signature: env.Signature}
	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, err
	}
	payloadHeader, err := unmarshalEnvelopsPayloadHeader(payload.Header)
	if err != nil {
		return nil, err
	}

	tx, err := utils.UnmarshalTransaction(payload.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "Envelope UnmarshalTransaction")
	}

	txActions := make([]*TransactionAction, 0, 1)
	for _, action := range tx.Actions {
		txAction, err := unmarshalTransactionAction(action)
		if err != nil {
			return nil, err
		}
		txActions = append(txActions, txAction)
	}
	envelope.Payload.Header = payloadHeader
	envelope.Payload.Transaction.Actions = txActions
	return envelope, nil
}

func unmarshalEnvelopsPayloadHeader(header *cb.Header) (*PayloadHeader, error) {
	payloadHeader := &PayloadHeader{}
	channelHeader, err := utils.UnmarshalChannelHeader(header.ChannelHeader)
	if err != nil {
		return nil, errors.WithMessage(err, "Payload UnmarshalChannelHeader")
	}

	payloadHeader.ChannelHeader = channelHeader
	sigHeader, err := unmarshalSignatureHeader(header.SignatureHeader)
	if err != nil {
		return nil, errors.WithMessage(err, "Payload UnmarshalSignatureHeader")
	}
	payloadHeader.SignatureHeader = *sigHeader
	return payloadHeader, nil
}

// unmarshalSignatureHeader 解析头部签名数据
func unmarshalSignatureHeader(sigBytes []byte) (*SignatureHeader, error) {
	sigHeader, err := utils.UnmarshalSignatureHeader(sigBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "Payload UnmarshalSignatureHeader")
	}
	id := msp.SerializedIdentity{}
	err = proto.Unmarshal(sigHeader.Creator, &id)
	if err != nil {
		return nil, err
	}
	return &SignatureHeader{
		Creator: &id,
		Nonce:   sigHeader.Nonce,
	}, nil
}

// unmarshalTransactionAction 解析具体交易部分
func unmarshalTransactionAction(action *pb.TransactionAction) (*TransactionAction, error) {
	txAction := &TransactionAction{}
	sigHeader, err := unmarshalSignatureHeader(action.Header)
	if err != nil {
		return nil, errors.WithMessage(err, "TransactionAction unmarshalSignatureHeader")
	}
	txAction.Header = sigHeader
	payload, err := utils.UnmarshalChaincodeActionPayload(action.Payload)
	if err != nil {
		return nil, errors.WithMessage(err, "TransactionAction UnmarshalChaincodeActionPayload")
	}
	endorserAction, err := unmarshalChaincodeEndorsedAction(payload.Action)
	if err != nil {
		return nil, err
	}
	txAction.Payload.Action = endorserAction
	propPayload, err := unmarshalChaincodeProposalPayload(payload.ChaincodeProposalPayload)
	if err != nil {
		return nil, err
	}
	txAction.Payload.ChaincodeProposalPayload = propPayload
	return txAction, nil
}

func unmarshalChaincodeProposalPayload(proposalPayload []byte) (*ChaincodeProposalPayload, error) {
	proposal := &ChaincodeProposalPayload{}
	payload, err := utils.UnmarshalChaincodeProposalPayload(proposalPayload)
	if err != nil {
		return nil, errors.WithMessage(err, "UnmarshalChaincodeProposalPayload")
	}
	proposal.TransientMap = payload.TransientMap
	input, err := utils.UnmarshalChaincodeInvocationSpec(payload.Input)
	if err != nil {
		return nil, errors.WithMessage(err, "UnmarshalChaincodeInvocationSpec")
	}
	proposal.Input = input.ChaincodeSpec
	return proposal, nil
}

// payload.Endorsements
// payload.ProposalResponsePayload
func unmarshalChaincodeEndorsedAction(payload *pb.ChaincodeEndorsedAction) (*ChaincodeEndorserAction, error) {
	endorserAction := &ChaincodeEndorserAction{
		Endorsements:            make([]Endorsement, len(payload.Endorsements)),
		ProposalResponsePayload: &ProposalResponsePayload{},
	}
	for i, e := range payload.Endorsements {
		endorserAction.Endorsements[i] = Endorsement{Signature: e.Signature}
		id, err := utils.UnmarshalSerializedIdentity(e.Endorser)
		if err != nil {

			return nil, errors.WithMessage(err, "unmarshal serialized identity")
		}
		endorserAction.Endorsements[i].Endorser = id
	}

	resp, err := utils.UnmarshalProposalResponsePayload(payload.ProposalResponsePayload)
	if err != nil {
		return nil, errors.WithMessage(err, "UnmarshalProposalResponsePayload")
	}
	endorserAction.ProposalResponsePayload.ProposalHash = resp.ProposalHash
	chaincodeAction, err := unmarshalChaincodeAction(resp.Extension)
	if err != nil {
		return nil, err
	}
	endorserAction.ProposalResponsePayload.Extension = *chaincodeAction
	return endorserAction, nil
}

func unmarshalChaincodeAction(extension []byte) (*ChaincodeAction, error) {
	action := &ChaincodeAction{}
	cc, _ := utils.UnmarshalChaincodeAction(extension)
	action.Response = cc.Response
	action.ChaincodeID = cc.ChaincodeId
	action.Events = cc.Events

	txRWSet, err := unmarshalTxRWSet(cc.Results)
	if err != nil {
		return nil, err
	}
	action.Results = txRWSet
	return action, nil
}

func unmarshalTxRWSet(data []byte) (*rwsetutil.TxRwSet, error) {
	return rwsetutil.ProtoUnmarshal(data)
}

