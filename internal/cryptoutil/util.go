package cryptoutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
)

func GetCertFromPEM(certBytes []byte) (*x509.Certificate, error) {
	return getCertFromPEM(certBytes)
}

func getCertFromPEM(idBytes []byte) (*x509.Certificate, error) {
	if idBytes == nil {
		return nil, errors.New("getCertFromPEM error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCertFromPEM error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	var cert *x509.Certificate
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCertFromPEM error: failed to parse x509 cert")
	}

	return cert, nil
}

func GetPrivateKeyFromPEM(raw []byte, pwd []byte) (Key, error) {
	k, err := pemToPrivateKey(raw, pwd)
	if err != nil {
		return nil, err
	}
	switch ks := k.(type) {
	case *ecdsa.PrivateKey:
		return &ecdsaPrivateKey{ks}, nil
	default:
		return nil, fmt.Errorf("unknown key type")
	}
}

// GetPEMFromPrivateKey 将私钥转化为byte
func GetPEMFromPrivateKey(key Key, pwd []byte) ([]byte, error) {
	pk, ok := key.(*ecdsaPrivateKey)
	if !ok {
		return nil, fmt.Errorf("not ecdsa private key")
	}
	return privateKeyToPEM(pk.priv, pwd)
}

func pemToPrivateKey(raw []byte, pwd []byte) (interface{}, error) {
	b, _ := pem.Decode(raw)
	if b == nil {
		return nil, fmt.Errorf("")
	}
	if x509.IsEncryptedPEMBlock(b) {
		// TODO
		if len(pwd) == 0 {
			return nil, fmt.Errorf("must have password")
		}
		return encryptPEMToPrivateKey(raw, pwd)
	}
	fmt.Println("=======================")
	derBytes := b.Bytes
	if k, err := x509.ParsePKCS1PrivateKey(derBytes); err == nil {
		return k, nil
	}

	if k, err := x509.ParsePKCS8PrivateKey(derBytes); err == nil {
		return k, nil
	}

	if k, err := x509.ParseECPrivateKey(derBytes); err == nil {
		return k, nil
	}
	return nil, errors.New("Invalid key type. The DER must contain an rsa.PrivateKey or ecdsa.PrivateKey")
}


func encryptPEMToPrivateKey(raw []byte, pwd []byte) (interface{}, error) {
	if len(raw) == 0 {
		return nil, errors.New("invalid PEM. it must be different from nil.")
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("failed decoding PEM. block must be different from nil. [% x]", raw)
	}
	if x509.IsEncryptedPEMBlock(block) {
		if len(pwd) == 0 {
			return nil, errors.New("Encrypted Key. Need a password")
		}

		decrypted, err := x509.DecryptPEMBlock(block, pwd)
		if err != nil {
			return nil, fmt.Errorf("Failed PEM decryption [%s]", err)
		}

		key, err := DERToPrivateKey(decrypted)
		if err != nil {
			return nil, err
		}
		return key, err
	}
	cert, err := DERToPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, err
}

// DERToPrivateKey unmarshals a der to private key
func DERToPrivateKey(der []byte) (key interface{}, err error) {

	if key, err = x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}

	if key, err = x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return
		default:
			return nil, errors.New("Found unknown private key type in PKCS#8 wrapping")
		}
	}

	if key, err = x509.ParseECPrivateKey(der); err == nil {
		return
	}

	return nil, errors.New("Invalid key type. The DER must contain an rsa.PrivateKey or ecdsa.PrivateKey")
}

func privateKeyToPEM(privateKey interface{}, pwd []byte) ([]byte, error) {
	switch k := privateKey.(type) {
	case *ecdsa.PrivateKey:
		if len(pwd) > 0 {
			return privateToEncryptoPEM(privateKey, pwd)
		}
		oidNamedCurve, ok := oidFromNamedCurve(k.Curve)
		if !ok {
			return nil, fmt.Errorf("")
		}
		privateKeyBytes := k.D.Bytes()
		paddedPrivateKey := make([]byte, (k.Curve.Params().N.BitLen()+7)/8)
		copy(paddedPrivateKey[len(paddedPrivateKey)-len(privateKeyBytes):], privateKeyBytes)
		asn1Bytes, err := asn1.Marshal(ecPrivateKey{
			Version:    1,
			PrivateKey: paddedPrivateKey,
			PublicKey:  asn1.BitString{Bytes: elliptic.Marshal(k.Curve, k.X, k.Y)},
		})
		if err != nil {
			return nil, fmt.Errorf("")
		}

		var pkcs8Key pkcs8Info
		pkcs8Key.Version = 0
		pkcs8Key.PrivateKeyAlgorithm = make([]asn1.ObjectIdentifier, 2)
		pkcs8Key.PrivateKeyAlgorithm[0] = oidPublicKeyECDSA
		pkcs8Key.PrivateKeyAlgorithm[1] = oidNamedCurve
		pkcs8Key.PrivateKey = asn1Bytes

		pkcs8Bytes, err := asn1.Marshal(pkcs8Key)

		return pem.EncodeToMemory(
			&pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: pkcs8Bytes,
			},
		), nil
	default:
		return nil, fmt.Errorf("Invalid key type:%v. It must be *ecdsa.PrivateKey or *rsa.PrivateKey", k)
	}
}


func privateToEncryptoPEM(privateKey interface{}, pwd []byte) ([]byte, error) {
	switch prk := privateKey.(type) {
	case *ecdsa.PrivateKey:
		if prk == nil {
			return nil, fmt.Errorf("Invalid ecdsa private key. It must be different from nil.")
		}
		raw, err := x509.MarshalECPrivateKey(prk)
		if err != nil {
			return nil, err
		}
		block, err := x509.EncryptPEMBlock(
			rand.Reader,
			"PRIVATE KEY",
			raw,
			pwd,
			x509.PEMCipherAES256)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(block), nil
	default:
		return nil, fmt.Errorf("Invalid key type. It must be *ecdsa.PrivateKey")
	}
}


// struct to hold info required for PKCS#8
type pkcs8Info struct {
	Version             int
	PrivateKeyAlgorithm []asn1.ObjectIdentifier
	PrivateKey          []byte
}

type ecPrivateKey struct {
	Version       int
	PrivateKey    []byte
	NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
	PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
}

var oidPublicKeyECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}

func oidFromNamedCurve(curve elliptic.Curve) (asn1.ObjectIdentifier, bool) {
	switch curve {
	case elliptic.P224():
		return asn1.ObjectIdentifier{1, 3, 132, 0, 33}, true
	case elliptic.P256():
		return asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}, true
	case elliptic.P384():
		return asn1.ObjectIdentifier{1, 3, 132, 0, 34}, true
	case elliptic.P521():
		return asn1.ObjectIdentifier{1, 3, 132, 0, 35}, true
	}
	return nil, false
}

// B64Decode base64 decodes a string
func B64Decode(str string) (buf []byte, err error) {
	return base64.StdEncoding.DecodeString(str)
}

func B64Encode(buf []byte) string {
	return base64.StdEncoding.EncodeToString(buf)
}

func GenECDSAToken(cert []byte, key Key, method, uri string, body []byte) (string, error) {
	b64body := B64Encode(body)
	b64cert := B64Encode(cert)
	b64uri := B64Encode([]byte(uri))
	payload := method + "." + b64uri + "." + b64body + "." + b64cert
	hasher := hasher{hash: sha256.New}
	return genecdsaToken(b64cert, payload, key, hasher)
}

func genecdsaToken(b64cert, payload string, key Key, hasher Hasher) (string, error) {
	signer, err := NewECDSASigner(key)
	if err != nil {
		return "", err
	}
	digest := hasher.Hash([]byte(payload))
	sig, err := signer.Sign(rand.Reader, digest, nil)
	if err != nil {
		return "", err
	}
	b64sig := B64Encode(sig)
	token := b64cert + "." + b64sig
	return token, nil
}

const (
	// NonceSize is the default NonceSize
	NonceSize = 24
)

// GetRandomBytes returns len random looking bytes
func GetRandomBytes(len int) ([]byte, error) {
	key := make([]byte, len)

	// TODO: rand could fill less bytes then len
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}

	return key, nil
}

// GetRandomNonce returns a random byte array of length NonceSize
func GetRandomNonce() ([]byte, error) {
	return GetRandomBytes(NonceSize)
}

func GetCertificateFromFile(certFile string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	return GetCertFromPEM(data)
}
