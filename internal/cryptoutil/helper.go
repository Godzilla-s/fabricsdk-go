package cryptoutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"strconv"
	"strings"
)


// SignerAlgo returns an X.509 signature algorithm from a crypto.Signer.
func SignerAlgo(priv crypto.Signer) x509.SignatureAlgorithm {
	switch pub := priv.Public().(type) {
	case *rsa.PublicKey:
		bitLength := pub.N.BitLen()
		switch {
		case bitLength >= 4096:
			return x509.SHA512WithRSA
		case bitLength >= 3072:
			return x509.SHA384WithRSA
		case bitLength >= 2048:
			return x509.SHA256WithRSA
		default:
			return x509.SHA1WithRSA
		}
	case *ecdsa.PublicKey:
		switch pub.Curve {
		case elliptic.P521():
			return x509.ECDSAWithSHA512
		case elliptic.P384():
			return x509.ECDSAWithSHA384
		case elliptic.P256():
			return x509.ECDSAWithSHA256
		default:
			return x509.ECDSAWithSHA1
		}
	default:
		return x509.UnknownSignatureAlgorithm
	}
}

// appendIf appends to a if s is not an empty string.
func appendIf(s string, a *[]string) {
	if s != "" {
		*a = append(*a, s)
	}
}

// OIDFromString creates an ASN1 ObjectIdentifier from its string representation
func OIDFromString(s string) (asn1.ObjectIdentifier, error) {
	var oid []int
	parts := strings.Split(s, ".")
	if len(parts) < 1 {
		return oid, fmt.Errorf("invalid OID string: %s", s)
	}
	for _, p := range parts {
		i, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid OID part %s", p)
		}
		oid = append(oid, i)
	}
	return oid, nil
}

// appendCAInfoToCSR appends user-defined extension to a CSR
func appendExtensionsToCSR(extensions []pkix.Extension, csr *x509.CertificateRequest) error {
	for _, extension := range extensions {
		csr.ExtraExtensions = append(csr.ExtraExtensions, extension)
	}
	return nil
}

// BasicConstraints CSR information RFC 5280, 4.2.1.9
type BasicConstraints struct {
	IsCA       bool `asn1:"optional"`
	MaxPathLen int  `asn1:"optional,default:-1"`
}

// appendCAInfoToCSR appends CAConfig BasicConstraint extension to a CSR
func appendCAInfoToCSR(reqConf *CAConfig, csr *x509.CertificateRequest) error {
	pathlen := reqConf.PathLength
	if pathlen == 0 && !reqConf.PathLenZero {
		pathlen = -1
	}
	val, err := asn1.Marshal(BasicConstraints{true, pathlen})

	if err != nil {
		return err
	}

	csr.ExtraExtensions = append(csr.ExtraExtensions, pkix.Extension{
		Id:       asn1.ObjectIdentifier{2, 5, 29, 19},
		Value:    val,
		Critical: true,
	})

	return nil
}
