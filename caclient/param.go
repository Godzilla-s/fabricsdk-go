package caclient

import "github.com/cloudflare/cfssl/signer"

type IdentityType string

const (
	ROLE_ID_PEER    IdentityType = "peer"
	ROLE_ID_ORDERER              = "orderer"
	ROLE_ID_USER                 = "client"
	ROLE_ID_ADMIN                = "admin"
)

func (id IdentityType) String() string {
	return string(id)
}

// EnrollmentRequest: request params for enroll
type EnrollmentRequest struct {
	Name    string
	Secret  string
	CAName  string
	Profile string
	Label   string
	// CSR     *cryptoutil.CSRInfo
	Type    string
	AttrReq  []*AttributeRequest
	CN      string
	Hosts    []string
}

type AttributeRequest struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional,omitempty"`
}

type EnrollmentRequestNet struct {
	signer.SignRequest
	CAName   string
	AttrReqs []*AttributeRequest `json:"attr_reqs,omitempty"`
}

// GetName returns the name of an attribute being requested
func (ar *AttributeRequest) GetName() string {
	return ar.Name
}

// IsRequired returns true if the attribute being requested is required
func (ar *AttributeRequest) IsRequired() bool {
	return !ar.Optional
}

type CAInfoResponseNet struct {
	// CAName is a unique name associated with fabric-ca-server's CA
	CAName string
	// Base64 encoding of PEM-encoded certificate chain
	CAChain string
	// Base64 encoding of Idemix issuer public key
	IssuerPublicKey string
	// Base64 encoding of PEM-encoded Idemix issuer revocation public key
	IssuerRevocationPublicKey string
	// Version of the server
	Version string
}

// EnrollmentResponseNet is the response to the /enroll request
type EnrollmentResponseNet struct {
	// Base64 encoded PEM-encoded ECert
	Cert string
	// The server information
	ServerInfo CAInfoResponseNet
}

// ####################  register request ######################
type RegistrationRequest struct {
	Name           string      `json:"id" help:"Unique name of the identity"`
	Type           string      `json:"type" def:"client" help:"Type of identity being registered (e.g. 'peer, app, user')"`
	Secret         string      `json:"secret,omitempty" mask:"password" help:"The enrollment secret for the identity being registered"`
	MaxEnrollments int         `json:"max_enrollments,omitempty" help:"The maximum number of times the secret can be reused to enroll (default CA's Max Enrollment)"`
	Affiliation    string      `json:"affiliation" help:"The identity's affiliation"`
	Attributes     []Attribute `json:"attrs,omitempty"`
	CAName         string      `json:"caname,omitempty" skip:"true"`
	Profile        string      `json:"profile,omitempty"`
}

// Attribute is a name and value pair
type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	ECert bool   `json:"ecert,omitempty"`
}

func NewRegisterRequest(name, secret string, idtype IdentityType) RegistrationRequest {
	req := RegistrationRequest{
		Name:name,
		Secret:secret,
		Type:string(idtype),
	}

	switch idtype {
	case ROLE_ID_PEER:
		req.Attributes = DefaultPeerAttr
	case ROLE_ID_ORDERER:
		req.Attributes = DefaultOrdererAttr
	case ROLE_ID_USER:
		req.Attributes = DefaultClientAttr
	case ROLE_ID_ADMIN:
		req.Attributes = DefaultAdminAttr
	}
	return req
}

func (req *RegistrationRequest) SetAttribute(attrKey, attrValue string) {
	req.Attributes = append(req.Attributes, Attribute{Name: attrKey, Value:attrValue, ECert:true})
}

// RegistrationResponse is a registration response
type RegistrationResponse struct {
	Secret string `json:"secret"`
}

type GetIdentityResponse struct {
	ID             string      `json:"id" skip:"true"`
	Type           string      `json:"type" def:"user"`
	Affiliation    string      `json:"affiliation"`
	Attributes     []Attribute `json:"attrs" mapstructure:"attrs" `
	MaxEnrollments int         `json:"max_enrollments" mapstructure:"max_enrollments"`
	CAName         string      `json:"caname,omitempty"`
}

type RemoveIdentityRequest struct {
	ID     string `skip:"true"`
	Force  bool   `json:"force"`
	CAName string `json:"caname,omitempty" skip:"true"`
}

type IdentityResponse struct {
	ID             string      `json:"id" skip:"true"`
	Type           string      `json:"type,omitempty"`
	Affiliation    string      `json:"affiliation"`
	Attributes     []Attribute `json:"attrs,omitempty" mapstructure:"attrs"`
	MaxEnrollments int         `json:"max_enrollments,omitempty" mapstructure:"max_enrollments"`
	Secret         string      `json:"secret,omitempty"`
	CAName         string      `json:"caname,omitempty"`
}

// ModifyIdentityRequest represents the request to modify an existing identity on the
// fabric-ca-server
type ModifyIdentityRequest struct {
	ID             string      `skip:"true"`
	Type           string      `json:"type" help:"Type of identity being registered (e.g. 'peer, app, user')"`
	Affiliation    string      `json:"affiliation" help:"The identity's affiliation"`
	Attributes     []Attribute `mapstructure:"attrs" json:"attrs"`
	MaxEnrollments int         `mapstructure:"max_enrollments" json:"max_enrollments" help:"The maximum number of times the secret can be reused to enroll"`
	Secret         string      `json:"secret,omitempty" mask:"password" help:"The enrollment secret for the identity"`
	CAName         string      `json:"caname,omitempty" skip:"true"`
}

// ###################### Affiliation Params #####################
type AddAffiliationRequest struct {
	Name   string `json:"name"`
	Force  bool   `json:"force"`
	CAName string `json:"caname,omitempty"`
}

type RemoveAffiliationRequest struct {
	Name   string
	Force  bool   `json:"force"`
	CAName string `json:"caname,omitempty"`
}

type GetAffiliationRequest struct {
	Name   string
	CAName string
}

type AffiliationResponse struct {
	AffiliationInfo `mapstructure:",squash"`
	CAName          string `json:"caname,omitempty"`
}

type AffiliationInfo struct {
	Name         string            `json:"name"`
	Affiliations []AffiliationInfo `json:"affiliations,omitempty"`
	Identities   []IdentityInfo    `json:"identities,omitempty"`
}

type IdentityInfo struct {
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Affiliation    string      `json:"affiliation"`
	Attributes     []Attribute `json:"attrs" mapstructure:"attrs"`
	MaxEnrollments int         `json:"max_enrollments" mapstructure:"max_enrollments"`
}

//##################### Default Attribute for register #########################
// default admin client attribute
var DefaultAdminAttr = []Attribute{
	{
		Name:  "hf.Registrar.Roles",
		Value: "client,orderer,peer,user",
	},
	{
		Name:  "hf.Registrar.DelegateRoles",
		Value: "client,orderer,peer,user",
	},
	{
		Name:  "hf.Registrar.Attributes",
		Value: "*",
	},
	{
		Name:  "hf.GenCRL",
		Value: "true",
	},
	{
		Name:  "hf.Revoker",
		Value: "true",
	},
	{
		Name:  "hf.AffiliationMgr",
		Value: "true",
	},
	{
		Name:  "role",
		Value: "admin",
		ECert: true,
	},
}

// default peer attribute
var DefaultPeerAttr = []Attribute{
	{
		Name:  "role",
		Value: "peer",
		ECert: true,
	},
}

// default orderer attribute
var DefaultOrdererAttr = []Attribute{
	{
		Name:  "role",
		Value: "orderer",
		ECert: true,
	},
}

var DefaultClientAttr = []Attribute{
	{
		Name:  "role",
		Value: "client",
		ECert: true,
	},
}

