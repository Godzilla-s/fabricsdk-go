package orderer

type Option func() string

func WithEndpoint(endpoint string) Option {
	return func() string {
		return endpoint
	}
}

