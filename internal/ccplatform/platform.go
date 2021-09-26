package ccplatform

import (
	"archive/tar"
	"fmt"
)

type Platform interface {
	Name() string
	ValidateCodePackage(code []byte) error
	GetDeploymentPayload(path string) ([]byte, error)
}

type PackageWriter interface {
	Write(name string, payload []byte, tw *tar.Writer) error
}

type PackageWriterWrapper func(name string, payload []byte, tw *tar.Writer) error

func (pw PackageWriterWrapper) Write(name string, payload []byte, tw *tar.Writer) error {
	return pw(name, payload, tw)
}

type Registry struct {
	Platforms     map[string]Platform
	PackageWriter PackageWriter
}

func NewRegistry(platformTypes ...Platform) *Registry{
	platforms := make(map[string]Platform)
	for _, platform := range platformTypes {
		if _, ok := platforms[platform.Name()]; ok {
			panic("Multiple platforms of the same name specified: " + platform.Name())
		}
		platforms[platform.Name()] = platform
	}
	r := &Registry{
		Platforms:     platforms,
		PackageWriter: PackageWriterWrapper(WriteBytesToPackage),
	}
	return r
}

func (r *Registry) GetDeploymentPayload(ccType, path string) ([]byte, error) {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return nil, fmt.Errorf("Unknown chaincodeType: %s", ccType)
	}
	return platform.GetDeploymentPayload(path)
}

func (r *Registry) NormalizePath(ccType, path string) (string, error) {
	platform, ok := r.Platforms[ccType]
	if !ok {
		return "", fmt.Errorf("unknown chaincodeType: %s", ccType)
	}
	if normalizer, ok := platform.(NormalizePather); ok {
		return normalizer.NormalizePath(path)
	}
	return path, nil
}

// NormalizerPather is an optional interface that can be implemented by
// platforms to modify the path stored in the chaincde ID.
type NormalizePather interface {
	NormalizePath(path string) (string, error)
}

// PlatformRegistry defines the interface to get the code bytes
// for a chaincode given the type and path
type PlatformRegistry interface {
	GetDeploymentPayload(path string) ([]byte, error)
	NormalizePath(path string) (string, error)
}


