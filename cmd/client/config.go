package main

type CommonConfig struct {
	// Required.
	ServerURL string

	// Required.
	AuthToken string

	// Optional.
	CryptoKey string
}

type CreateTopicConfig struct {
	CommonConfig

	// Required.
	Topic string
}
