package channel

import (
	"crypto/x509"
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/hyperledger/fabric-config/configtx"
	"github.com/hyperledger/fabric-config/configtx/membership"
	"github.com/hyperledger/fabric-config/configtx/orderer"
	"time"
)

const (
	CSCC             = "cscc"
	CSCC_JoinChannel = "JoinChain"
	CSCC_GetChainInfo = "GetChainInfo"
	CSCC_GetChannels = "GetChannels"

	QSCC_GetBlockByHash = "GetBlockByHash"
	QSCC_GetBlockByNumber = "GetBlockByNumber"
	QSCC_GetBlockByTxID = "GetBlockByTxID"
	QSCC_GetTransactionByTxID = "GetTransactionByID"
)

var (
	standardApplicationChannelPoliciesV2 = map[string]configtx.Policy{
		configtx.ReadersPolicyKey: {
			Type: configtx.ImplicitMetaPolicyType,
			Rule: "ANY Readers",
		},
		configtx.WritersPolicyKey: {
			Type: configtx.ImplicitMetaPolicyType,
			Rule: "ANY Writers",
		},
		configtx.AdminsPolicyKey: {
			Type: configtx.ImplicitMetaPolicyType,
			Rule: "MAJORITY Admins",
		},
		configtx.EndorsementPolicyKey: {
			Type: configtx.ImplicitMetaPolicyType,
			Rule: "MAJORITY Endorsement",
		},
		configtx.LifecycleEndorsementPolicyKey: {
			Type: configtx.ImplicitMetaPolicyType,
			Rule: "MAJORITY Endorsement",
		},
	}
)
type ChannelConfig struct {
	OrdererType string
	// Batch Timeout: The amount of time to wait before creating a batch
	BatchTimeout time.Duration
	BatchSize    orderer.BatchSize
	Option       orderer.EtcdRaftOptions
}

type Organization struct {
	// 组织类型， peer, orderer
	Type protoutil.Organization_Type
	// 组织MSP ID
	ID string
	// 组织名称
	Name string
	// 组织根证书
	RootCA *x509.Certificate
	// tls 根证书
	TLSRootCA *x509.Certificate
	// 组织admin证书
	AdminCA *x509.Certificate
	// 锚节点，只有组织为peer时
	AnchorPeers []AnchorPeer
	// 排序组织节点信息
	OrdererConsenters []Consenter
}

type AnchorPeer struct {
	Host string
	Port int
}

// Consenter etcdraft集群中orderer节点信息
type Consenter struct {
	Host          string
	Port          int
	ServerTLSCert x509.Certificate
	ClientTLSCert x509.Certificate
}

func (c ChannelConfig) GetBatchSize() orderer.BatchSize {
	batchSize := c.BatchSize
	if batchSize.AbsoluteMaxBytes == 0 {
		batchSize.AbsoluteMaxBytes = 10 * 1024 * 1024 // 10MB
	}
	if batchSize.MaxMessageCount == 0 {
		batchSize.MaxMessageCount = 500
	}
	if batchSize.PreferredMaxBytes == 0 {
		batchSize.PreferredMaxBytes = 2 * 1024 * 1024 // 2MB
	}
	return batchSize
}

func (o Organization) CreateOrganization() (configtx.Organization, error) {
	org := configtx.Organization{
		Name:     o.Name,
		Policies: getPeerOrgStandardRWPolicies(o.ID),
	}
	switch o.Type {
	case protoutil.Organization_PEER:
		org.Policies = getOrdererOrgStandardRWPolicy(o.ID)
	case protoutil.Organization_ORDERER:
		org.Policies = getPeerOrgStandardRWPolicies(o.ID)
	default:
		return org, fmt.Errorf("organization type is required")
	}

	//var adminCA *x509.Certificate
	org.MSP = createMSP(o.ID, o.RootCA, o.TLSRootCA, nil)
	org.AnchorPeers = make([]configtx.Address, len(o.AnchorPeers))
	for i, ap := range o.AnchorPeers {
		org.AnchorPeers[i] = configtx.Address{Host: ap.Host, Port: ap.Port}
	}
	if len(o.OrdererConsenters) > 0 {
		for _, order := range o.OrdererConsenters {
			org.OrdererEndpoints = append(org.OrdererEndpoints, fmt.Sprintf("%s:%d", order.Host, order.Port))
		}
	}
	return org, nil
}

func getPeerOrgStandardRWPolicies(mspId string) map[string]configtx.Policy {
	policies := map[string]configtx.Policy{
		configtx.ReadersPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.admin', '%s.peer', '%s.client')", mspId, mspId, mspId),
		},
		configtx.WritersPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.admin', '%s.client')", mspId, mspId),
		},
		configtx.AdminsPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.admin')", mspId),
		},
		configtx.EndorsementPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.peer')", mspId),
		},
	}
	return policies
}

func getOrdererOrgStandardRWPolicy(mspid string) map[string]configtx.Policy {
	policies := map[string]configtx.Policy{
		configtx.ReadersPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.member')", mspid),
		},
		configtx.WritersPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.member')", mspid),
		},
		configtx.AdminsPolicyKey: {
			Type: configtx.SignaturePolicyType,
			Rule: fmt.Sprintf("OR('%s.admin')", mspid),
		},
	}
	return policies
}

func createMSP(mspID string, rootCA, tlsRootCA, adminCA *x509.Certificate) configtx.MSP {
	msp := configtx.MSP{
		Name:         mspID,
		RootCerts:    []*x509.Certificate{rootCA},
		TLSRootCerts: []*x509.Certificate{tlsRootCA},
		CryptoConfig: membership.CryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
		NodeOUs: membership.NodeOUs{
			Enable: true,
			ClientOUIdentifier: membership.OUIdentifier{
				Certificate:                  rootCA,
				OrganizationalUnitIdentifier: "client",
			},
			PeerOUIdentifier: membership.OUIdentifier{
				Certificate:                  rootCA,
				OrganizationalUnitIdentifier: "peer",
			},
			AdminOUIdentifier: membership.OUIdentifier{
				Certificate:                  rootCA,
				OrganizationalUnitIdentifier: "admin",
			},
			OrdererOUIdentifier: membership.OUIdentifier{
				Certificate:                  rootCA,
				OrganizationalUnitIdentifier: "orderer",
			},
		},
	}
	if adminCA != nil {
		msp.Admins = []*x509.Certificate{adminCA}
	}
	return msp
}