package chaincode

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/godzilla-s/fabricsdk-go/internal/ccplatform"
	"github.com/godzilla-s/fabricsdk-go/internal/ccplatform/golang"
	lb "github.com/hyperledger/fabric-protos-go/peer/lifecycle"
	"github.com/pkg/errors"
	"io/ioutil"
)

type ChaincodeInstaller interface {
	GetInstalledChaincode() (*lb.InstallChaincodeArgs, error)
}

type chaincodePackageCode struct {
	packageData []byte
}

// GetChaincodeInstallerFromPackage 从打好的安装包中安装
func GetChaincodeInstallerFromPackage(pkgBytes []byte) ChaincodeInstaller {
	return &chaincodePackageCode{
		packageData: pkgBytes,
	}
}

func (c *chaincodePackageCode) GetInstalledChaincode() (*lb.InstallChaincodeArgs, error) {
	return &lb.InstallChaincodeArgs{ChaincodeInstallPackage: c.packageData}, nil
}

type chaincodePackageFile struct {
	pkgFile string
}

// GetChaincodeInstallerFromPkgFile 打包的链码文件安装
func GetChaincodeInstallerFromPkgFile(pkgFile string) ChaincodeInstaller {
	return &chaincodePackageFile{
		pkgFile: pkgFile,
	}
}

// GetChaincodeInstallerFromSource 源码安装
func GetChaincodeInstallerFromSource(path, label, typ string) ChaincodeInstaller {
	// TODO
	var packager ccplatform.PlatformRegistry
	switch typ {
	case "GOLANG":
		packager = &golang.Platform{}
	}
	return &chaincodeFromPath{packager: packager, lang: typ, label: label,ccPath: path}
}

// GetChaincodeInstallerFromGitRepo 从git安装源码
func GetChaincodeInstallerFromGitRepo(git string) ChaincodeInstaller {
	return nil
}

func (pkg *chaincodePackageFile) GetInstalledChaincode() (*lb.InstallChaincodeArgs, error) {
	pkgBytes, err := ioutil.ReadFile(pkg.pkgFile)
	if err != nil {
		return nil, err
	}

	return &lb.InstallChaincodeArgs{ChaincodeInstallPackage: pkgBytes}, nil
}


type chaincodeFromPath struct {
	ccPath string
	label  string
	lang  string
	packager ccplatform.PlatformRegistry
}

// PackageMetadata holds the path and type for a chaincode package
type PackageMetadata struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

func toJSON(path, ccType, label string) ([]byte, error) {
	metadata := &PackageMetadata{
		Path:  path,
		Type:  ccType,
		Label: label,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal chaincode package metadata into JSON")
	}

	return metadataBytes, nil
}

func writeBytesToPackage(tw *tar.Writer, name string, payload []byte) error {
	err := tw.WriteHeader(&tar.Header{
		Name: name,
		Size: int64(len(payload)),
		Mode: 0100644,
	})
	if err != nil {
		return err
	}

	_, err = tw.Write(payload)
	if err != nil {
		return err
	}

	return nil
}

func (cc *chaincodeFromPath) getTarPackage() ([]byte, error) {
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	normalizedPath, err := cc.packager.NormalizePath(cc.ccPath)
	if err != nil {
		return nil, err
	}
	metadataBytes, err := toJSON(normalizedPath, cc.lang, cc.ccPath)
	if err != nil {
		return nil, err
	}
	err = writeBytesToPackage(tw, "metadata.json", metadataBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error writing package metadata to tar")
	}
	codeBytes, err := cc.packager.GetDeploymentPayload(cc.ccPath)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting chaincode bytes")
	}
	codePackageName := "code.tar.gz"

	err = writeBytesToPackage(tw, codePackageName, codeBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error writing package code bytes to tar")
	}

	err = tw.Close()
	if err == nil {
		err = gw.Close()
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create tar for chaincode")
	}
	return payload.Bytes(), nil
}

func (cc *chaincodeFromPath) GetInstalledChaincode() (*lb.InstallChaincodeArgs, error) {
	chaincodeBytes, err := cc.getTarPackage()
	if err != nil {
		return nil, err
	}
	// TODO: 想要安装是指定组织签名
	return &lb.InstallChaincodeArgs{ChaincodeInstallPackage: chaincodeBytes}, nil
}

