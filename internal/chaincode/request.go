package chaincode

import "time"

// ApproveChaincodeRequest used for approve chaincode
type ApproveChaincodeRequest struct {
	PackageID                string
	Name                     string
	TxID                     string
	Version                  string
	Sequence                 int64
	SignPolicy               string
	ChannelConfigPolicy      string
	EndorserPlugin           string
	ValidationPlugin         string
	ValidationParameterBytes []byte
	CollectionConfig   		 []byte
	InitRequired             bool
	WaitForEvent             bool
	WaitForEventTimeout      time.Duration
}

// CommitChaincodeRequest used for commit chaincode
type CommitChaincodeRequest struct {
	TxID                string
	Name                string
	Version             string
	Sequence            int64
	SignPolicy          string
	ChannelConfigPolicy string
	EndorsementPlugin   string
	ValidationPlugin    string
	ValidationParameter []byte
	InitRequired        bool
	CollectionConfig    []byte
	WaitForEvent        bool
	WaitForEventTimeout time.Duration
}

type LifecycleArgs struct {
	Name                string
	Version             string
	Sequence            int64
	EndorsementPlugin   string
	ValidationPlugin    string
	ValidationParameter []byte
	InitRequired        bool
	// Collections string
}

// CheckCommitReadinessRequest
type CheckCommitReadinessRequest struct {
	Sequence             int64
	Name                 string
	Version              string
	EndorsementPlugin    string
	ValidationPlugin     string
	ValidationParameter  []byte
	InitRequired         bool
	ChannelID string
}

type QueryCommittedChaincodeRequest struct {
	LifecycleArgs
}
