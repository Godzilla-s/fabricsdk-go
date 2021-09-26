package caclient

import "github.com/godzilla-s/fabricsdk-go/internal/cryptoutil"

type KeyStore interface {
	// 获取根证书
	GetRootCert()  []byte
	// 获取签名证书
	GetSignCert()  []byte
	// 获取签名私钥
	GetKey()       cryptoutil.Key
	// 获取私钥原始数据
	GetRawKey(pwd []byte) ([]byte, error)
}

type keystore struct {
	rootCert  []byte
	signCert  []byte
	key       cryptoutil.Key
}

func (ks keystore) GetRootCert() []byte {
	return ks.rootCert
}

func (ks keystore) GetSignCert() []byte {
	return ks.signCert
}

func (ks keystore) GetKey() cryptoutil.Key {
	return ks.key
}

func (ks keystore) GetRawKey(pwd []byte) ([]byte, error) {
	return cryptoutil.GetPEMFromPrivateKey(ks.key, pwd)
}
