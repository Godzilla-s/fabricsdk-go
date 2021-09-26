package contract

import (
	"github.com/godzilla-s/fabricsdk-go/internal/chaincode"
	"time"
)

type Option func(req *chaincode.ChaincodeSpec) *chaincode.ChaincodeSpec


func WithInit() Option {
	return func(req *chaincode.ChaincodeSpec) *chaincode.ChaincodeSpec {
		req.IsInit = true
		return req
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(req *chaincode.ChaincodeSpec) *chaincode.ChaincodeSpec {
		req.Timeout = timeout
		return req
	}
}

func WithTransient(tmap string) Option {
	return func(req *chaincode.ChaincodeSpec) *chaincode.ChaincodeSpec {
		req.Transient = tmap
		return req
	}
}

func WithLang(lang string) Option {
	return func(req *chaincode.ChaincodeSpec) *chaincode.ChaincodeSpec {
		req.Lang = lang
		return req
	}
}
