package contract

import (
	"github.com/godzilla-s/fabricsdk-go/internal/chaincode"
	"github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"
	"github.com/pkg/errors"
)

type Contract struct {
	signer  cryptoutil.Signer
	channelID string
	name   string
	lang  string
	version string
	cf *chaincode.CommonFactory
}

func New(signer cryptoutil.Signer, name, version, channelID string, impl *chaincode.CommonFactory, options ...Option) (*Contract, error) {
	_, err := chaincode.QueryCommitted(signer, impl.Committer, channelID, chaincode.WithName(name))
	if err != nil {
		return nil, errors.WithMessagef(err, "fail to get chaincode %s:%s that commit on channel %s", name, version, channelID)
	}

	req := &chaincode.ChaincodeSpec{}
	for _, opt := range options {
		req = opt(req)
	}
	c := &Contract{
		signer: signer,
		channelID: channelID,
		name: name,
		version: version,
		cf: impl,
		lang: req.Lang,
	}
	return c, nil
}

// Invoke 调用合约
func (c Contract) Invoke(args [][]byte, opts ...Option) (*chaincode.Response, error) {
	spec := &chaincode.ChaincodeSpec{Lang: c.lang}
	for _, opt := range opts {
		spec = opt(spec)
	}
	spec.Name = c.name
	spec.Version = c.version
	spec.Args = args
	spec.Lang = c.lang
	return chaincode.Invoke(c.signer, c.cf, *spec, c.channelID)
}

// Query 查询合约
func (c Contract) Query(args [][]byte) (*chaincode.Response, error) {
	spec := chaincode.ChaincodeSpec{
		Name: c.name,
		Version: c.version,
		Args: args,
	}
	return chaincode.Query(c.signer, c.cf, spec, c.channelID)
}


// SendTransaction 只发送交易，不产生区块
func (c Contract) SendTransaction(args [][]byte, opts ...Option) (*chaincode.ProcessProposalResult, error) {
	spec := &chaincode.ChaincodeSpec{Lang: c.lang}
	for _, opt := range opts {
		spec = opt(spec)
	}
	spec.Name = c.name
	spec.Version = c.version
	spec.Args = args
	// TODO
	return chaincode.SendTransaction(c.signer, c.cf, *spec, c.channelID)
}

