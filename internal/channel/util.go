package channel

import (
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/blockutil"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
)

func SignUpdateConfig(signer cryptoutil.Signer, update []byte) (*cb.ConfigSignature, error) {
	signedHeader, err := signer.NewSignatureHeader()
	if err != nil {
		return nil, err
	}

	header, err := proto.Marshal(signedHeader)
	if err != nil {
		return nil, fmt.Errorf("marshaling signature header: %v", err)
	}

	configSignature := &cb.ConfigSignature{
		SignatureHeader: header,
	}
	data := concatenateBytes(configSignature.SignatureHeader, update)
	signature, err := signer.Sign(data)
	if err != nil {
		return nil, err
	}
	configSignature.Signature = signature
	return configSignature, nil
}

// concatenateBytes combines multiple arrays of bytes, for signatures or digests over multiple fields.
func concatenateBytes(data ...[]byte) []byte {
	var res []byte
	for i := range data {
		res = append(res, data[i]...)
	}
	return res
}

func getBlockConfig(lastConfigBlock *cb.Block) (*cb.Config, error) {
	configBlock, err := blockutil.UnmarshalConfig(lastConfigBlock)
	if err != nil {
		return nil, err
	}
	return configBlock.Data[0].Payload.Data.Config, nil
}

func CreateUpdateEnvelope(update []byte, sigs map[string][]byte, channeliD string) (*UpdateEnvelope, error) {
	updateEnvelope := &UpdateEnvelope{update: update, channelID: channeliD, signatures: make(map[string]*cb.ConfigSignature)}
	for name, sigBytes := range sigs {
		sig := &cb.ConfigSignature{}
		err := proto.Unmarshal(sigBytes, sig)
		if err != nil {
			return nil, err
		}
		updateEnvelope.signatures[name] = sig
	}
	return updateEnvelope, nil
}
