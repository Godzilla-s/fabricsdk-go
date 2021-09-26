package cryptoutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
)

type ecdsaPrivateKey struct {
	priv *ecdsa.PrivateKey
}

func (k *ecdsaPrivateKey) Bytes() ([]byte, error) {
	return nil, errors.New("Not supported.")
}

func (k *ecdsaPrivateKey) SKI() []byte {
	if k.priv == nil {
		return nil
	}

	// Marshall the public key
	raw := elliptic.Marshal(k.priv.Curve, k.priv.PublicKey.X, k.priv.PublicKey.Y)
	// Hash it
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (k *ecdsaPrivateKey) Symmetric() bool {
	return false
}

func (k *ecdsaPrivateKey) Private() bool {
	return true
}

func (k *ecdsaPrivateKey) PublicKey() (Key, error) {
	return &ecdsaPublicKey{&k.priv.PublicKey}, nil
}

type ecdsaPublicKey struct {
	puk *ecdsa.PublicKey
}

func (pub *ecdsaPublicKey) Bytes() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(pub.puk)
}

func (pub *ecdsaPublicKey) SKI() []byte {
	if pub.puk == nil {
		return nil
	}
	// Marshall the public key
	raw := elliptic.Marshal(pub.puk.Curve, pub.puk.X, pub.puk.Y)
	// Hash it
	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (pub *ecdsaPublicKey) Symmetric() bool {
	return false
}

func (pub *ecdsaPublicKey) Private() bool {
	return false
}

func (pub *ecdsaPublicKey) PublicKey() (Key, error) {
	return pub, nil
}

type ecdsaKeyGenerator struct {
	curve elliptic.Curve
}

func (ecdsaGen *ecdsaKeyGenerator) KeyGen() (Key, error) {
	prik, err := ecdsa.GenerateKey(ecdsaGen.curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ecdsaPrivateKey{prik}, nil
}

type ecdsaSigner struct {
	prik Key
	pk   interface{}
}

func (ecdsasigner *ecdsaSigner) Public() crypto.PublicKey {
	return ecdsasigner.pk
}

func (ecdsaSigner *ecdsaSigner) Sign(reader io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	pk, ok := ecdsaSigner.prik.(*ecdsaPrivateKey)
	if !ok {
		return nil, fmt.Errorf("not ecdsaPrivateKey")
	}
	r, s, err := ecdsa.Sign(rand.Reader, pk.priv, digest)
	if err != nil {
		return nil, err
	}

	puk := pk.priv.PublicKey
	s, _, err = ToLowS(&puk, s)
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(ECDSASignature{r, s})
}

type ECDSASignature struct {
	R, S *big.Int
}

var curveHalfOrders = map[elliptic.Curve]*big.Int{
	elliptic.P224(): new(big.Int).Rsh(elliptic.P224().Params().N, 1),
	elliptic.P256(): new(big.Int).Rsh(elliptic.P256().Params().N, 1),
	elliptic.P384(): new(big.Int).Rsh(elliptic.P384().Params().N, 1),
	elliptic.P521(): new(big.Int).Rsh(elliptic.P521().Params().N, 1),
}

func IsLowS(k *ecdsa.PublicKey, s *big.Int) (bool, error) {
	halfOrder, ok := curveHalfOrders[k.Curve]
	if !ok {
		return false, fmt.Errorf("not support curve")
	}
	return s.Cmp(halfOrder) != 1, nil
}

func ToLowS(k *ecdsa.PublicKey, s *big.Int) (*big.Int, bool, error) {
	lowS, err := IsLowS(k, s)
	if err != nil {
		return nil, false, err
	}

	if !lowS {
		s.Sub(k.Params().N, s)
		return s, true, nil
	}

	return s, false, nil
}

func newEcdsaSigner(key Key) (crypto.Signer, error) {
	if !key.Private() {
		return  nil, fmt.Errorf("key must be private key")
	}
	puk, err := key.PublicKey()
	if err != nil {
		return nil, err
	}
	pukBytes, err := puk.Bytes()
	if err != nil {
		return nil, err
	}
	pk, err := x509.ParsePKIXPublicKey(pukBytes)
	if err != nil {
		return nil, err
	}
	return &ecdsaSigner{prik: key, pk: pk}, nil
}

func NewECDSASigner(key Key) (crypto.Signer, error) {
	if !key.Private() {
		return nil, fmt.Errorf("key must be private key")
	}

	puk, err := key.PublicKey()
	if err != nil {
		return nil, err
	}

	pukBytes, err := puk.Bytes()
	if err != nil {
		return nil, err
	}

	pk, err := x509.ParsePKIXPublicKey(pukBytes)
	if err != nil {
		return nil, err
	}
	return &ecdsaSigner{prik: key, pk: pk}, nil
}


