package cryptoutil

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

type Signer interface {
	Serialize() ([]byte, error)
	Sign(msp []byte) ([]byte, error)
	NewSignatureHeader() (*cb.SignatureHeader, error)
	GetMSPId() string
}

func GetSignedProposal(prop *pb.Proposal, signer Signer) (*pb.SignedProposal, error) {
	propBytes, err := proto.Marshal(prop)
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(propBytes)
	if err != nil {
		return nil, err
	}

	return &pb.SignedProposal{ProposalBytes: propBytes, Signature: signature}, nil
}

type myCryptoSigner struct {
	opt      string
	priKey   Key
	mspID    string
	signCert *x509.Certificate
	hashOpt  string
}

func (s *myCryptoSigner) getSigner() (crypto.Signer, error) {
	return newEcdsaSigner(s.priKey)
}

func (s *myCryptoSigner) Sign(msg []byte) ([]byte, error) {
	digest, err := Hash(msg, s.hashOpt)
	if err != nil {
		return nil, err
	}
	sign, err := s.getSigner()
	if err != nil {
		return nil, err
	}
	return sign.Sign(rand.Reader, digest, nil)
}

func (s *myCryptoSigner) Serialize() ([]byte, error) {
	pblock := &pem.Block{Bytes: s.signCert.Raw, Type: "CERTIFICATE"}
	pemBytes := pem.EncodeToMemory(pblock)
	id := &msp.SerializedIdentity{Mspid: s.mspID, IdBytes: pemBytes}
	bytes, err := proto.Marshal(id)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (s *myCryptoSigner) NewSignatureHeader() (*cb.SignatureHeader, error)  {
	creator, err := s.Serialize()
	if err != nil {
		return nil, err
	}
	nonce, err := GetRandomNonce()
	if err != nil {
		return nil, err
	}
	sh := &cb.SignatureHeader{}
	sh.Creator = creator
	sh.Nonce = nonce
	return sh, nil
}

func (s *myCryptoSigner) GetMSPId() string {
	return s.mspID
}
