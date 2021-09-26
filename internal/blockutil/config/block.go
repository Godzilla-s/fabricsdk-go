package config

import (
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

type Block struct {
	Header  *cb.BlockHeader
	Metadata *cb.BlockMetadata
	Data     []*Envelope
}

// .data.data[0]
type Envelope struct {
	Payload   *Payload
	Signature []byte
}

// .data.data[0].payload
type Payload struct {
	Data           *ConfigEnvelope
	Header         *Header
}

// .data.data[0].payload.data
type ConfigEnvelope struct {
	Config     *cb.Config
	LastUpdate *LastUpdate
}

// .data.data[0].payload.data.last_update
type LastUpdate struct {
	Payload   *LastConfigPayload
	Signature []byte
}

// .data.data[0].payload.data.last_update.payload
type LastConfigPayload struct {
	Data    *ConfigUpdateEnvelope
	Header  *Header
}

// .data.data[0].payload.data.last_update.payload.data
type ConfigUpdateEnvelope struct {
	ConfigUpdate *cb.ConfigUpdate  // .data.data[0].payload.data.last_update.payload.data.config_update
	Signatures  []*ConfigSignature //
}

type ConfigSignature struct {
	SignatureHeader *SignatureHeader
	Signature  []byte
}

type SignatureHeader struct {
	Creator *msp.SerializedIdentity
	Nonce   []byte
}

type Header struct {
	ChannelHeader   *cb.ChannelHeader
	SignatureHeader SignatureHeader
}

// ---------------------------------------------
// unmarshal config block
// ---------------------------------------------
func UnmarshalConfigPayload(data []byte) (*Payload, error) {
	envPayload, err := utils.UnmarshalPayload(data)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal payload")
	}

	configEnv, err := unmarshalPayloadConfigEnvelope(envPayload.Data)
	if err != nil {
		return nil, err
	}

	header, err := unmarshalHeader(envPayload.Header)
	if err != nil {
		return nil, err
	}
	payload := &Payload{
		Data: configEnv,
		Header: header,
	}
	return payload, nil
}

func unmarshalPayloadConfigEnvelope(data []byte) (*ConfigEnvelope, error) {
	env := &cb.ConfigEnvelope{}
	err := proto.Unmarshal(data, env)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal config")
	}
	var lastUpdate *LastUpdate
	if env.LastUpdate != nil {
		lastUpdate, err = unmarshalLastUpdate(env.LastUpdate)
		if err != nil {
			return nil, err
		}
	}
	configEnv := &ConfigEnvelope{
		Config: env.Config,
		LastUpdate: lastUpdate,
	}
	return configEnv, nil
}

func unmarshalLastUpdate(update *cb.Envelope) (*LastUpdate, error) {
	payload, err := utils.UnmarshalPayload(update.Payload)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal payload")
	}

	lastConfigPayload, err := unmarshalLastConfigPayload(payload)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal last config payload")
	}
	lastUpdate := &LastUpdate{
		Payload: lastConfigPayload,
		Signature: update.Signature,
	}
	return lastUpdate, nil
}

func unmarshalLastConfigPayload(env *cb.Payload) (*LastConfigPayload, error) {
	updateConfig, err := unmarshalConfigUpdateEnvelope(env.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal config update")
	}

	header, err := unmarshalHeader(env.Header)
	if err != nil {
		return nil, err
	}
	lastConfigPayload := &LastConfigPayload{
		Data: updateConfig,
		Header: header,
	}
	return lastConfigPayload, nil
}

func unmarshalConfigUpdateEnvelope(data []byte) (*ConfigUpdateEnvelope, error) {
	updateEnv, err := utils.UnmarshalConfigUpdateEnvelope(data)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal config update envelope")
	}
	configUpdate := &cb.ConfigUpdate{}
	err = proto.Unmarshal(updateEnv.ConfigUpdate, configUpdate)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal config update")
	}

	signatures := make([]*ConfigSignature, len(updateEnv.Signatures))
	for i, sig := range updateEnv.Signatures {
		sigHeader, err := unmarshalSignatureHeader(sig.SignatureHeader)
		if err != nil {
			return nil, errors.WithMessage(err, "unmarshal signature header")
		}
		signatures[i] = &ConfigSignature{
			SignatureHeader: sigHeader,
			Signature: sig.Signature,
		}
	}

	lastConfigUpdate := &ConfigUpdateEnvelope{
		ConfigUpdate: configUpdate,
		Signatures: signatures,
	}

	return lastConfigUpdate, nil
}

func unmarshalSignatureHeader(data []byte) (*SignatureHeader, error) {
	sigHeader, err := utils.UnmarshalSignatureHeader(data)
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

func unmarshalHeader(header *cb.Header) (*Header, error) {
	h := &Header{}
	channelHeader, err := utils.UnmarshalChannelHeader(header.ChannelHeader)
	if err != nil {
		return nil, errors.WithMessage(err, "Payload UnmarshalChannelHeader")
	}

	h.ChannelHeader = channelHeader
	sigHeader, err := unmarshalSignatureHeader(header.SignatureHeader)
	if err != nil {
		return nil, errors.WithMessage(err, "Payload UnmarshalSignatureHeader")
	}
	h.SignatureHeader = *sigHeader
	return h, nil
}
