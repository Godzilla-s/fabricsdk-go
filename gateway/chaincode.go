package gateway

import (
	"context"
	"github.com/godzilla-s/fabricsdk-go/gateway/protoutil"
	"github.com/godzilla-s/fabricsdk-go/internal/chaincode"
	"github.com/godzilla-s/fabricsdk-go/internal/chaincode/contract"
	"github.com/pkg/errors"
)

func ChaincodeInstall(ctx context.Context, req *protoutil.ChaincodeInstallRequest) (*protoutil.ChaincodeInstallResponse, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}

	pClients, err := createPeerClients(req.Peers)
	if err != nil {
		return nil, err
	}

	var chaincodeInstaller chaincode.ChaincodeInstaller
	switch req.Chaincode.Mode {
	case protoutil.ChaincodePackage_FROM_PACKAGE_BYTES:
		pkgBytes := req.Chaincode.Chaincode.GetPkgBytes()
		chaincodeInstaller = chaincode.GetChaincodeInstallerFromPackage(pkgBytes)
	case protoutil.ChaincodePackage_FROM_PACKAGE_FILE:
		pkgFile := req.Chaincode.Chaincode.GetFile()
		chaincodeInstaller = chaincode.GetChaincodeInstallerFromPkgFile(pkgFile)
	case protoutil.ChaincodePackage_FROM_SOURCE_CODE:
		// TODO
		chaincodeInstaller = chaincode.GetChaincodeInstallerFromSource("", "", "")
	case protoutil.ChaincodePackage_FROM_GIT_REPO:
		// TODO
		chaincodeInstaller = chaincode.GetChaincodeInstallerFromGitRepo("")
	}

	return chaincode.Install(signer, pClients, chaincodeInstaller)
}

// ApproveChaincode 授信链码
func ChaincodeApprove(ctx context.Context, req *protoutil.ChaincodeApproveRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}
	commonFactory, err := createCommonFactory(req.Committer, []*protoutil.Peer{req.Committer}, req.Orderer)
	if err != nil {
		return nil, err
	}
	definition := &chaincode.ApproveChaincodeRequest{
		Name: req.Definition.Name,
		Version: req.Definition.Version,
		PackageID: req.PackageId,
		Sequence: req.Definition.Sequence,
		EndorserPlugin: req.Definition.EndorsePlugin,
		ValidationPlugin: req.Definition.ValidatePlugin,
		ValidationParameterBytes: req.Definition.ValidateParams,
	}
	resp, err := chaincode.Approve(signer, commonFactory, definition, req.ChannelId)
	if err != nil {
		return nil, err
	}
	return &protoutil.Response{Status: resp.Response.Status}, nil
}

// CommitChaincode 提交确认链码
func ChaincodeCommit(ctx context.Context, req *protoutil.ChaincodeCommitRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, err
	}

	commonFactory, err := createCommonFactory(nil, req.Endorsers, req.Orderer)
	if err != nil {
		return nil, err
	}

	definition := &chaincode.CommitChaincodeRequest{
		Name: req.Definition.Name,
		Version: req.Definition.Version,
		Sequence: req.Definition.Sequence,
		EndorsementPlugin: req.Definition.EndorsePlugin,
		InitRequired: req.Definition.InitRequired,
		ValidationPlugin: req.Definition.ValidatePlugin,
		ValidationParameter: req.Definition.ValidateParams,
	}
	resp, err := chaincode.Commit(signer, commonFactory, definition, req.ChannelId)
	if err != nil {
		return nil, err
	}
	return &protoutil.Response{Status: resp.Response.Status}, nil
}

// InvokeChaincode 调用链码
func ChaincodeInvoke(ctx context.Context, req *protoutil.ContractInvokeRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, errors.WithMessage(err, "get signer")
	}
	commonFactory, err := createCommonFactory(req.Committer, req.Endorsers, req.Orderer)
	if err != nil {
		return nil, errors.WithMessage(err, "get common factory")
	}
	c, err := contract.New(signer, req.Args.Name, req.Args.Version, req.ChannelId, commonFactory)
	if err != nil {
		return nil, errors.WithMessage(err, "new contract")
	}
	resp, err := c.Invoke(req.Args.Args)
	if err != nil {
		return nil, errors.WithMessage(err, "invoke")
	}
	return &protoutil.Response{Status: resp.Response.Status, Payload: []byte(resp.TxID)}, nil
}

// QueryChaincode 查询链码
func ChaincodeQuery(ctx contract.Contract, req *protoutil.ContractQueryRequest) (*protoutil.Response, error) {
	signer, err := createSigner(req.Signer)
	if err != nil {
		return nil, errors.WithMessage(err, "get signer")
	}
	commonFactory, err := createCommonFactory(req.Committer, nil, req.Orderer)
	if err != nil {
		return nil, errors.WithMessage(err, "get common factory")
	}
	c, err := contract.New(signer, req.Args.Name, req.Args.Version, req.ChannelId, commonFactory)
	if err != nil {
		return nil, errors.WithMessage(err, "new contract")
	}
	resp, err := c.Query(req.Args.Args)
	if err != nil {
		return nil, errors.WithMessage(err, "invoke")
	}

	return &protoutil.Response{Status: resp.Response.Status, Message: resp.Response.Message, Payload: resp.Response.Payload}, nil
}