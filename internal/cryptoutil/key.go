package cryptoutil

type Key interface {
	Bytes() ([]byte, error)
	SKI() []byte
	Symmetric() bool
	Private() bool
	PublicKey() (Key, error)
}

type KeyGenerator interface {
	KeyGen() (Key, error)
}

