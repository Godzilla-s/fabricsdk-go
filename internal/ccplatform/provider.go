package ccplatform


import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"strings"
	"unicode"
)

// CCPackage encapsulates a chaincode package which can be
//    raw ChaincodeDeploymentSpec
//    SignedChaincodeDeploymentSpec
// Attempt to keep the interface at a level with minimal
// interface for possible generalization.
type CCPackage interface {
	//InitFromBuffer initialize the package from bytes
	InitFromBuffer(buf []byte) (*ChaincodeData, error)

	// InitFromFS gets the chaincode from the filesystem (includes the raw bytes too)
	//InitFromFS(ccname string, ccversion string) ([]byte, *pb.ChaincodeDeploymentSpec, error)

	// PutChaincodeToFS writes the chaincode to the filesystem
	//PutChaincodeToFS() error

	// GetDepSpec gets the ChaincodeDeploymentSpec from the package
	GetDepSpec() *peer.ChaincodeDeploymentSpec

	// GetDepSpecBytes gets the serialized ChaincodeDeploymentSpec from the package
	GetDepSpecBytes() []byte

	// ValidateCC validates and returns the chaincode deployment spec corresponding to
	// ChaincodeData. The validation is based on the metadata from ChaincodeData
	// One use of this method is to validate the chaincode before launching
	ValidateCC(ccdata *ChaincodeData) error

	// GetPackageObject gets the object as a protos.Message
	GetPackageObject() proto.Message

	// GetChaincodeData gets the ChaincodeData
	GetChaincodeData() *ChaincodeData

	// GetId gets the fingerprint of the chaincode based on package computation
	GetId() []byte
}

// ExtractSignedCCDepSpec extracts the messages from the envelope
func ExtractSignedCCDepSpec(env *common.Envelope) (*common.ChannelHeader, *peer.SignedChaincodeDeploymentSpec, error) {
	p := &common.Payload{}
	err := proto.Unmarshal(env.Payload, p)
	if err != nil {
		return nil, nil, err
	}
	if p.Header == nil {
		return nil, nil, errors.New("channel header cannot be nil")
	}
	ch := &common.ChannelHeader{}
	err = proto.Unmarshal(p.Header.ChannelHeader, ch)
	if err != nil {
		return nil, nil, err
	}

	sp := &peer.SignedChaincodeDeploymentSpec{}
	err = proto.Unmarshal(p.Data, sp)
	if err != nil {
		return nil, nil, err
	}

	return ch, sp, nil
}


//-------- ChaincodeData is stored on the LSCC -------

// ChaincodeData defines the datastructure for chaincodes to be serialized by protos
// Type provides an additional check by directing to use a specific package after instantiation
// Data is Type specific (see CDSPackage and SignedCDSPackage)
type ChaincodeData struct {
	// Name of the chaincode
	Name string `protobuf:"bytes,1,opt,name=name"`

	// Version of the chaincode
	Version string `protobuf:"bytes,2,opt,name=version"`

	// Escc for the chaincode instance
	Escc string `protobuf:"bytes,3,opt,name=escc"`

	// Vscc for the chaincode instance
	Vscc string `protobuf:"bytes,4,opt,name=vscc"`

	// Policy endorsement policy for the chaincode instance
	Policy []byte `protobuf:"bytes,5,opt,name=policy,proto3"`

	// Data data specific to the package
	Data []byte `protobuf:"bytes,6,opt,name=data,proto3"`

	// Id of the chaincode that's the unique fingerprint for the CC This is not
	// currently used anywhere but serves as a good eyecatcher
	Id []byte `protobuf:"bytes,7,opt,name=id,proto3"`

	// InstantiationPolicy for the chaincode
	InstantiationPolicy []byte `protobuf:"bytes,8,opt,name=instantiation_policy,proto3"`
}

// CCName returns the name of this chaincode (the name it was put in the ChaincodeRegistry with).
func (cd *ChaincodeData) CCName() string {
	return cd.Name
}

// Hash returns the hash of the chaincode.
func (cd *ChaincodeData) Hash() []byte {
	return cd.Id
}

// CCVersion returns the version of the chaincode.
func (cd *ChaincodeData) CCVersion() string {
	return cd.Version
}

// Validation returns how to validate transactions for this chaincode.
// The string returned is the name of the validation method (usually 'vscc')
// and the bytes returned are the argument to the validation (in the case of
// 'vscc', this is a marshaled pb.VSCCArgs message).
func (cd *ChaincodeData) Validation() (string, []byte) {
	return cd.Vscc, cd.Policy
}

// Endorsement returns how to endorse proposals for this chaincode.
// The string returns is the name of the endorsement method (usually 'escc').
func (cd *ChaincodeData) Endorsement() string {
	return cd.Escc
}

// implement functions needed from protos.Message for protos's mar/unmarshal functions

// Reset resets
func (cd *ChaincodeData) Reset() { *cd = ChaincodeData{} }

// String converts to string
func (cd *ChaincodeData) String() string { return proto.CompactTextString(cd) }

// ProtoMessage just exists to make protos happy
func (*ChaincodeData) ProtoMessage() {}


// isPrintable is used by CDSPackage and SignedCDSPackage validation to
// detect garbage strings in unmarshaled protos fields where printable
// characters are expected.
func isPrintable(name string) bool {
	notASCII := func(r rune) bool {
		return !unicode.IsPrint(r)
	}
	return strings.IndexFunc(name, notASCII) == -1
}

// GetCCPackage tries each known package implementation one by one
// till the right package is found
func GetCCPackage(buf []byte) (CCPackage, error) {
	// try raw CDS
	cds := &CDSPackage{}
	if ccdata, err := cds.InitFromBuffer(buf); err != nil {
		cds = nil
	} else {
		err = cds.ValidateCC(ccdata)
		if err != nil {
			cds = nil
		}
	}

	// try signed CDS
	scds := &SignedCDSPackage{}
	if ccdata, err := scds.InitFromBuffer(buf); err != nil {
		scds = nil
	} else {
		err = scds.ValidateCC(ccdata)
		if err != nil {
			scds = nil
		}
	}

	if cds != nil && scds != nil {
		// Both were unmarshaled successfully, this is exactly why the approach of
		// hoping protos fails for bad inputs is fatally flawed.
		//ccproviderLogger.Errorf("Could not determine chaincode package type, guessing SignedCDS")
		return scds, nil
	}

	if cds != nil {
		return cds, nil
	}

	if scds != nil {
		return scds, nil
	}

	return nil, errors.New("could not unmarshal chaincode package to CDS or SignedCDS")
}

