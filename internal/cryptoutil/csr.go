package cryptoutil

import (
	"crypto/elliptic"
	"fmt"
	"github.com/cloudflare/cfssl/csr"
)

type CSRInfo struct {
	CN           string     `json:"CN"`
	Names        []csr.Name `json:"names,omitempty"`
	Hosts        []string   `json:"hosts,omitempty"`
	CA           *csr.CAConfig
	KeyRequest   *BasicKeyRequest
	SerialNumber string
}

type BasicKeyRequest struct {
	Algo string `json:"algo" yaml:"algo" help:"Specify key algorithm"`
	Size int    `json:"size" yaml:"size" help:"Specify key size"`
}

func GenerateKey(req *CSRInfo, id string) ([]byte, Key, error) {
	generator := ecdsaKeyGenerator{curve: elliptic.P256()}
	key, err := generator.KeyGen()
	if err != nil {
		return nil, nil, fmt.Errorf("KeyGen: %v", err)
	}
	signer, err := NewECDSASigner(key)
	if err != nil {
		return nil, nil, fmt.Errorf("NewECDSASigner: %v", err)
	}

	cr := newCertificateRequest(req)
	//cr.CN = id
	certPEM, err := csr.Generate(signer, cr)
	if err != nil {
		return nil, nil, fmt.Errorf("Generate: %v", err)
	}
	return certPEM, key, nil
}

func newCertificateRequest(req *CSRInfo) *csr.CertificateRequest {
	var cr = &csr.CertificateRequest{}
	cr.CN = req.CN
	if req.Hosts != nil && len(req.Hosts) > 0 {
		cr.Hosts = req.Hosts
	}
	if req.Names != nil && len(req.Names) > 0 {
		cr.Names = req.Names
	}
	if req.CA != nil {
		cr.CA = req.CA
	}

	if req != nil && req.KeyRequest != nil {
		cr.KeyRequest = newCfsslBasicKeyRequest(req.KeyRequest)
	}
	return cr
}

func newCfsslBasicKeyRequest(bkr *BasicKeyRequest) *csr.KeyRequest {
	return &csr.KeyRequest{A: bkr.Algo, S: bkr.Size}
}

