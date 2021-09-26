package chaincode

import "time"

type Option func(v interface{}) interface{}

func WithEndorsePlugin(plugin string) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.EndorserPlugin = plugin
		case *CommitChaincodeRequest:
			v.EndorsementPlugin = plugin
		}
		return obj
	}
}

// WithValidate
func WithValidatePlugin(plugin string) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.ValidationPlugin = plugin
		case *CommitChaincodeRequest:
			v.ValidationPlugin = plugin
		}
		return obj
	}
}

// WithEndorse 设置背书
func WithSignPolicy(policy string) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.SignPolicy = policy
		case *CommitChaincodeRequest:
			v.SignPolicy = policy
		}
		return obj
	}
}

// WithChannelPolicy
func WithChannelPolicy(policy string) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.ChannelConfigPolicy = policy
		case *CommitChaincodeRequest:
			v.ChannelConfigPolicy = policy
		}
		return obj
	}
}

// WithCollectionConfig
func WithCollectionConfig(config []byte) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.CollectionConfig = config
		case *CommitChaincodeRequest:
			v.CollectionConfig = config
		}
		return obj
	}
}

// WithSequence
func WithSequence(seqNum int64) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.Sequence = seqNum
		case *CommitChaincodeRequest:
			v.Sequence = seqNum
		case *CheckCommitReadinessRequest:
			v.Sequence = seqNum
		}
		return obj
	}
}

// WithTimeout
func WithTimeout(timeout time.Duration) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.WaitForEventTimeout = timeout
		case *CommitChaincodeRequest:
			v.WaitForEventTimeout = timeout
		}
		return obj
	}
}

// WithInitRequired
func WithInitRequired() Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *ApproveChaincodeRequest:
			v.InitRequired = true
		case *CommitChaincodeRequest:
			v.InitRequired = true
		case *CheckCommitReadinessRequest:
			v.InitRequired = true
		}
		return obj
	}
}

func WithName(name string) Option {
	return func(obj interface{}) interface{} {
		switch v := obj.(type) {
		case *QueryCommittedChaincodeRequest:
			v.Name = name
		}
		return obj
	}
}

