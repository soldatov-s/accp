package rabbitmq

import "time"

type PublisherConfig struct {
	DSN           string
	BackoffPolicy []time.Duration
	ExchangeName  string
}

func (pc *PublisherConfig) Merge(target *PublisherConfig) *PublisherConfig {
	result := &PublisherConfig{
		DSN:           pc.DSN,
		BackoffPolicy: pc.BackoffPolicy,
		ExchangeName:  pc.ExchangeName,
	}

	if target.DSN != "" {
		result.DSN = target.DSN
	}

	if len(target.BackoffPolicy) > 0 {
		result.BackoffPolicy = target.BackoffPolicy
	}

	if target.ExchangeName != "" {
		result.ExchangeName = target.ExchangeName
	}

	return result
}
