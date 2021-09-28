package cryptoutil

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"time"
)

// Subject contains the information that should be used to override the
// subject information when signing a certificate.
type Subject struct {
	CN           string
	Names        []Name `json:"names"`
	SerialNumber string
}

type OID asn1.ObjectIdentifier

// Extension represents a raw extension to be included in the certificate.  The
// "value" field must be hex encoded.
type Extension struct {
	ID       OID `json:"id"`
	Critical bool       `json:"critical"`
	Value    string     `json:"value"`
}

// SignRequest stores a signature request, which contains the hostname,
// the CSR, optional subject information, and the signature profile.
//
// Extensions provided in the signRequest are copied into the certificate, as
// long as they are in the ExtensionWhitelist for the signer's policy.
// Extensions requested in the CSR are ignored, except for those processed by
// ParseCertificateRequest (mainly subjectAltName).
type SignRequest struct {
	Hosts       []string    `json:"hosts"`
	Request     string      `json:"certificate_request"`
	Subject     *Subject    `json:"subject,omitempty"`
	Profile     string      `json:"profile"`
	CRLOverride string      `json:"crl_override"`
	Label       string      `json:"label"`
	Serial      *big.Int    `json:"serial,omitempty"`
	Extensions  []Extension `json:"extensions,omitempty"`
	// If provided, NotBefore will be used without modification (except
	// for canonicalization) as the value of the notBefore field of the
	// certificate. In particular no backdating adjustment will be made
	// when NotBefore is provided.
	NotBefore time.Time
	// If provided, NotAfter will be used without modification (except
	// for canonicalization) as the value of the notAfter field of the
	// certificate.
	NotAfter time.Time
	// If ReturnPrecert is true a certificate with the CT poison extension
	// will be returned from the Signer instead of attempting to retrieve
	// SCTs and populate the tbsCert with them itself. This precert can then
	// be passed to SignFromPrecert with the SCTs in order to create a
	// valid certificate.
	ReturnPrecert bool

	// Arbitrary metadata to be stored in certdb.
	Metadata map[string]interface{} `json:"metadata"`
}

// A Name contains the SubjectInfo fields.
type Name struct {
	C            string            `json:"C,omitempty" yaml:"C,omitempty"`   // Country
	ST           string            `json:"ST,omitempty" yaml:"ST,omitempty"` // State
	L            string            `json:"L,omitempty" yaml:"L,omitempty"`   // Locality
	O            string            `json:"O,omitempty" yaml:"O,omitempty"`   // OrganisationName
	OU           string            `json:"OU,omitempty" yaml:"OU,omitempty"` // OrganisationalUnitName
	E            string            `json:"E,omitempty" yaml:"E,omitempty"`
	SerialNumber string            `json:"SerialNumber,omitempty" yaml:"SerialNumber,omitempty"`
	OID          map[string]string `json:"OID,omitempty", yaml:"OID,omitempty"`
}

// A KeyRequest contains the algorithm and key size for a new private key.
type KeyRequest struct {
	A string `json:"algo" yaml:"algo"`
	S int    `json:"size" yaml:"size"`
}

// CAConfig is a section used in the requests initialising a new CA.
type CAConfig struct {
	PathLength  int    `json:"pathlen" yaml:"pathlen"`
	PathLenZero bool   `json:"pathlenzero" yaml:"pathlenzero"`
	Expiry      string `json:"expiry" yaml:"expiry"`
	Backdate    string `json:"backdate" yaml:"backdate"`
}

// A CertificateRequest encapsulates the API interface to the
// certificate request functionality.
type CertificateRequest struct {
	CN           string           `json:"CN" yaml:"CN"`
	Names        []Name           `json:"names" yaml:"names"`
	Hosts        []string         `json:"hosts" yaml:"hosts"`
	KeyRequest   *KeyRequest      `json:"key,omitempty" yaml:"key,omitempty"`
	CA           *CAConfig        `json:"ca,omitempty" yaml:"ca,omitempty"`
	SerialNumber string           `json:"serialnumber,omitempty" yaml:"serialnumber,omitempty"`
	Extensions   []pkix.Extension `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	CRL          string           `json:"crl_url,omitempty" yaml:"crl_url,omitempty"`
}

// Name returns the PKIX name for the request.
func (cr *CertificateRequest) Name() (pkix.Name, error) {
	var name pkix.Name
	name.CommonName = cr.CN

	for _, n := range cr.Names {
		appendIf(n.C, &name.Country)
		appendIf(n.ST, &name.Province)
		appendIf(n.L, &name.Locality)
		appendIf(n.O, &name.Organization)
		appendIf(n.OU, &name.OrganizationalUnit)
		for k, v := range n.OID {
			oid, err := OIDFromString(k)
			if err != nil {
				return name, err
			}
			name.ExtraNames = append(name.ExtraNames, pkix.AttributeTypeAndValue{Type: oid, Value: v})
		}
		if n.E != "" {
			name.ExtraNames = append(name.ExtraNames, pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}, Value: n.E})
		}
	}
	name.SerialNumber = cr.SerialNumber
	return name, nil
}