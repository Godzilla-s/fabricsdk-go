package cryptoutil

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"net/mail"
	"net/url"
)

// ResponseMessage implements the standard for response errors and
// messages. A message has a code and a string message.
type ResponseMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Response implements the CloudFlare standard for API
// responses.
type Response struct {
	Success  bool              `json:"success"`
	Result   interface{}       `json:"result"`
	Errors   []ResponseMessage `json:"errors"`
	Messages []ResponseMessage `json:"messages"`
}

type CSRInfo struct {
	CN           string     `json:"CN"`
	Names        []Name `json:"names,omitempty"`
	Hosts        []string   `json:"hosts,omitempty"`
	CA           *CAConfig
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
	certPEM, err := generateCSR(signer, cr)
	if err != nil {
		return nil, nil, fmt.Errorf("generate error: %v", err)
	}
	return certPEM, key, nil
}

func newCertificateRequest(req *CSRInfo) *CertificateRequest {
	var cr = &CertificateRequest{}
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

func newCfsslBasicKeyRequest(bkr *BasicKeyRequest) *KeyRequest {
	return &KeyRequest{A: bkr.Algo, S: bkr.Size}
}

// replace calling github.com/cloudflare/cfssl/csr
func generateCSR(priv crypto.Signer, req *CertificateRequest) (csr []byte, err error) {
	sigAlgo := SignerAlgo(priv)
	if sigAlgo == x509.UnknownSignatureAlgorithm {
		return nil, errors.New("unknown signed algorithm")
	}

	subj, err := req.Name()
	if err != nil {
		return nil, err
	}

	var tpl = x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: sigAlgo,
	}

	for i := range req.Hosts {
		if ip := net.ParseIP(req.Hosts[i]); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else if email, err := mail.ParseAddress(req.Hosts[i]); err == nil && email != nil {
			tpl.EmailAddresses = append(tpl.EmailAddresses, email.Address)
		} else if uri, err := url.ParseRequestURI(req.Hosts[i]); err == nil && uri != nil {
			tpl.URIs = append(tpl.URIs, uri)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, req.Hosts[i])
		}
	}

	tpl.ExtraExtensions = []pkix.Extension{}

	if req.CA != nil {
		err = appendCAInfoToCSR(req.CA, &tpl)
		if err != nil {
			err = errors.Wrapf(err, "") // TODO
			return
		}
	}

	if req.Extensions != nil {
		err = appendExtensionsToCSR(req.Extensions, &tpl)
		if err != nil {
			err = errors.Wrapf(err, "")
			return
		}
	}

	csr, err = x509.CreateCertificateRequest(rand.Reader, &tpl, priv)
	if err != nil {
		err = errors.Wrapf(err, "")
		return
	}
	block := pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csr,
	}

	csr = pem.EncodeToMemory(&block)
	return
}