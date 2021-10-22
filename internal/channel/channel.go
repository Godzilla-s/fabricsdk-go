package channel

import (
	"fmt"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/utils"
	"github.com/hyperledger/fabric-config/configtx"
	"github.com/hyperledger/fabric-config/configtx/orderer"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
	"io/ioutil"
	"time"
)

const (
	FABRIC_VERSION_2_0 = "V2_0"
	ORDERER_CONSEN

)
type ChannelEnvelope interface {
	CreateEnvelope() (*cb.Envelope, error)
	ChannelID() string
}

type channelFile struct {
	channelTxFile string
	channelID string
}

func NewChannelFromFile(channelID, channelTxFile string) *channelFile {
	return &channelFile{channelID: channelID, channelTxFile: channelTxFile}
}

func (ch *channelFile) CreateEnvelope() (*cb.Envelope, error) {
	cftx, err := ioutil.ReadFile(ch.channelTxFile)
	if err != nil {
		return nil, err
	}

	return utils.UnmarshalEnvelope(cftx)
}

func (ch *channelFile) ChannelID() string {
	return ch.channelID
}

type applicationChannel struct {
	appChannel configtx.Channel
	channelID  string
}

func NewApplicationChannel(channel configtx.Channel, channelID string) (ChannelEnvelope, error) {
	return &applicationChannel{appChannel: channel, channelID: channelID}, nil
}

func (app *applicationChannel) CreateEnvelope() (*cb.Envelope, error) {
	configCreate, err := configtx.NewMarshaledCreateChannelTx(app.appChannel, app.channelID)
	if err != nil {
		return nil, err
	}

	return configtx.NewEnvelope(configCreate)
}

func (app *applicationChannel) ChannelID() string {
	return app.channelID
}

type channelBytes struct {
	data   []byte
	channelID string
}

func NewChannelFromBytes(channelID string, data []byte) ChannelEnvelope {
	return &channelBytes{channelID: channelID, data: data}
}

func (cb *channelBytes) CreateEnvelope() (*cb.Envelope, error) {
	return utils.UnmarshalEnvelope(cb.data)
}

func (cb *channelBytes) ChannelID() string {
	return cb.channelID
}

// CreateApplicationChannel 创建一个应用通道
func CreateApplicationChannel(channelID, consortiumName string, orgs []Organization) (ChannelEnvelope, error) {
	channelOrgs := make([]configtx.Organization, len(orgs))
	for i, org := range orgs {
		channelOrg, err := org.CreateOrganization()
		if err != nil {
			return nil, err
		}
		channelOrgs[i] = channelOrg
	}
	app := configtx.Application{
		Organizations: channelOrgs,
		Capabilities:  []string{FABRIC_VERSION_2_0}, // support fabric v2.x
		Policies:      standardApplicationChannelPoliciesV2,
	}
	appChannel := configtx.Channel{
		Consortium:  consortiumName,
		Application: app,
	}

	return NewApplicationChannel(appChannel, channelID)
}

func CreateSystemGenesisBlock(ordererOrg Organization, peerOrgs []Organization, consortiumName, sysChannelID string) (*cb.Block, error) {

	orderer := configtx.Orderer{
		OrdererType: orderer.ConsensusTypeEtcdRaft,
		BatchTimeout: 2*time.Second,
		Capabilities: []string{"V2_0"},
		BatchSize: orderer.BatchSize{
			MaxMessageCount: 10,
			AbsoluteMaxBytes: 99*1024*1024,
			PreferredMaxBytes: 512*1024,
		},
		Policies: map[string]configtx.Policy{
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
			configtx.BlockValidationPolicyKey: {
				Type: configtx.ImplicitMetaPolicyType,
				Rule: "ANY Writers",
			},
		},
		Organizations: []configtx.Organization{},
		State: orderer.ConsensusStateNormal,
	}

	consortium := configtx.Consortium{
		Name: consortiumName,
	}
	for _, peerOrg := range peerOrgs {
		org, err := peerOrg.CreateOrganization()
		if err != nil {
			return nil, err
		}
		consortium.Organizations = append(consortium.Organizations, org)
	}
	channelOrg := configtx.Channel{
		Orderer: orderer,
		Capabilities: []string{"V2_0"},
		Policies: map[string]configtx.Policy{
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
		},
		Consortiums: []configtx.Consortium{consortium},
	}
	return configtx.NewSystemChannelGenesisBlock(channelOrg, sysChannelID)
}

// UpdateEnvelope
type UpdateEnvelope struct {
	signatures map[string]*cb.ConfigSignature
	update     []byte
	channelID  string
}

func (c *UpdateEnvelope) GetUpdates() []byte {
	return c.update
}

// SignBy 签名
func (c *UpdateEnvelope) SignBy(signer cryptoutil.Signer) error {
	if _, ok := c.signatures[signer.GetMSPId()]; ok {
		return errors.Errorf("%s has signed updated config", signer.GetMSPId())
	}
	sig, err := SignUpdateConfig(signer, c.update)
	if err != nil {
		return err
	}
	c.signatures[signer.GetMSPId()] = sig
	return nil
}

func (c *UpdateEnvelope) CreateEnvelope() (*cb.Envelope, error) {
	var signatures []*cb.ConfigSignature
	for _, sig := range c.signatures {
		signatures = append(signatures, sig)
	}
	return configtx.NewEnvelope(c.update, signatures...)
}

func (c *UpdateEnvelope) ChannelID() string {
	return c.channelID
}

// ChannelAddOrg 通道加入组织
func ChannelAddOrg(lastConfigBlock *cb.Block, newOrg Organization, channelID string) (*UpdateEnvelope, error) {
	config, err := getBlockConfig(lastConfigBlock)
	if err != nil {
		return nil, err
	}
	configTx := configtx.New(config)
	application := configTx.Application()
	org := application.Organization(newOrg.Name)
	if org != nil {
		return nil, fmt.Errorf("organization %s exist in channel %s", newOrg.Name, channelID)
	}

	newOrgConfig, err := newOrg.CreateOrganization()
	if err != nil {
		return nil, err
	}

	err = application.SetOrganization(newOrgConfig)
	if err != nil {
		return nil, err
	}
	update, err := configTx.ComputeMarshaledUpdate(channelID)
	if err != nil {
		return nil, err
	}
	return &UpdateEnvelope{
		update:     update,
		channelID:  channelID,
		signatures: make(map[string]*cb.ConfigSignature),
	}, nil
}


// ChannelRemoveOrg 通道删除组织
func ChannelRemoveOrg(lastConfigBlock *cb.Block, orgName, channelID string) (*UpdateEnvelope, error) {
	config, err := getBlockConfig(lastConfigBlock)
	if err != nil {
		return nil, err
	}
	configTx := configtx.New(config)
	application := configTx.Application()
	if application == nil {
		return nil, fmt.Errorf("empty application")
	}

	org := application.Organization(orgName)
	if org == nil {
		return nil, fmt.Errorf("organization %s not exist in channel %s", orgName, channelID)
	}
	application.RemoveOrganization(orgName)
	update, err := configTx.ComputeMarshaledUpdate(channelID)
	if err != nil {
		return nil, err
	}

	return &UpdateEnvelope{
		update:     update,
		channelID:  channelID,
		signatures: make(map[string]*cb.ConfigSignature),
	}, nil
}


// ConsortiumAddOrg 联盟加入组织
func ConsortiumAddOrg(lastConfigBlock *cb.Block, newOrg Organization, consortiumName, channelID string) (*UpdateEnvelope, error) {
	config, err := getBlockConfig(lastConfigBlock)
	if err != nil {
		return nil, err
	}
	configTx := configtx.New(config)
	//err = addPeerOrgToConsortium(&configTx, consortiumName, newOrg)
	//if err != nil {
	//	return nil, err
	//}

	// origin
	newOrgConfig, err := newOrg.CreateOrganization()
	if err != nil {
		return nil, err
	}
	cg := configTx.Consortium(consortiumName)
	if cg == nil {
		// TODO: create new consortiumName and add to config
		// consortiums := config.Consortiums()
		return nil, errors.Errorf("not found consortiumName '%s' in config", consortiumName)
	}

	if cg.Organization(newOrg.Name) != nil {
		return nil, errors.Errorf("organization %s exist in consortiumName %s", newOrg.Name, consortiumName)
	}

	err = cg.SetOrganization(newOrgConfig)
	if err != nil {
		return nil, err
	}

	update, err := configTx.ComputeMarshaledUpdate(channelID)
	if err != nil {
		return nil, err
	}
	return &UpdateEnvelope{update: update, channelID: channelID, signatures: make(map[string]*cb.ConfigSignature)}, nil
}

// ConsortiumRemoveOrg 联盟删除组织
func ConsortiumRemoveOrg(lastConfigBlock *cb.Block, orgName, consortium, channelID string) (*UpdateEnvelope, error) {
	config, err := getBlockConfig(lastConfigBlock)
	if err != nil {
		return nil, err
	}
	configTx := configtx.New(config)
	cg := configTx.Consortium(consortium)
	if cg == nil {
		return nil, errors.Errorf("not found consortium %s", consortium)
	}

	consortiumOrg := cg.Organization(orgName)
	if consortiumOrg == nil {
		return nil, errors.Errorf("not found org %s exist in consortium organizations", orgName)
	}

	cg.RemoveOrganization(orgName)
	update, err := configTx.ComputeMarshaledUpdate(channelID)
	if err != nil {
		return nil, err
	}
	return &UpdateEnvelope{
		update:     update,
		channelID:  channelID,
		signatures: make(map[string]*cb.ConfigSignature)}, nil
}