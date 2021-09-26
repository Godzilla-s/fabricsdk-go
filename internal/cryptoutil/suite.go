package cryptoutil

import "crypto/x509"

type CryptoSuite interface {
	NewSigner() (Signer, error)
	GetCreator() ([]byte, error)
	GetMSPID() string
}

type myCryptoSuite struct {
	privKey  Key
	signCert *x509.Certificate
	mspID    string
	hashOpt string
}

func (cs myCryptoSuite) NewSigner() (Signer, error) {
	if cs.hashOpt == "" {
		cs.hashOpt = SHA2_256
	}
	return &myCryptoSigner{
		priKey: cs.privKey,
		signCert: cs.signCert,
		mspID: cs.mspID,
		hashOpt: cs.hashOpt,
	}, nil
}

func (cs myCryptoSuite) GetCreator() ([]byte, error) {
	signer, err := cs.NewSigner()
	if err != nil {
		return nil, err
	}
	return signer.Serialize()
}

func (cs myCryptoSuite) GetMSPID() string {
	return cs.mspID
}

func GetMyCryptoSuiteFromBytes(keyBytes, certBytes []byte, mspid string) (CryptoSuite, error) {
	signCert, err := getCertFromPEM(certBytes)
	if err != nil {
		return nil, err
	}
	priKey, err := GetPrivateKeyFromPEM(keyBytes, nil)
	if err != nil {
		return nil, err
	}
	return &myCryptoSuite{
		privKey: priKey,
		signCert: signCert,
		mspID: mspid,
	}, nil
}