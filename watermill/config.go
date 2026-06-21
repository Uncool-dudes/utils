package watermill

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

// SASLConfig holds SASL authentication settings.
type SASLConfig struct {
	Enable    bool   `json:"enable"`
	Mechanism string `json:"mechanism" validate:"omitempty,oneof=PLAIN SCRAM-SHA-256 SCRAM-SHA-512"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// TLSConfig holds TLS settings for Kafka connections.
type TLSConfig struct {
	Enable             bool   `json:"enable"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CACert             string `json:"ca_cert"`
	ClientCert         string `json:"client_cert"`
	ClientKey          string `json:"client_key"`
}

// RetryConfig holds Kafka publisher retry settings.
type RetryConfig struct {
	MaxRetries        int     `json:"max_retries"`
	InitialIntervalMs int     `json:"initial_interval_ms"`
	Multiplier        float64 `json:"multiplier"`
}

// Config holds Watermill/Kafka connection settings.
type Config struct {
	Brokers          []string    `json:"brokers"           validate:"required,min=1"`
	ConsumerGroup    string      `json:"consumer_group"    validate:"required"`
	PoisonQueueTopic string      `json:"poison_queue_topic"`
	SASL             SASLConfig  `json:"sasl"`
	TLS              TLSConfig   `json:"tls"`
	Retry            RetryConfig `json:"retry"`
}

// Defaults provides sane out-of-the-box Config values.
var Defaults = Config{
	Retry: RetryConfig{
		MaxRetries:        3,
		InitialIntervalMs: 100,
		Multiplier:        2.0,
	},
}

func (c Config) poisonTopic() string {
	if c.PoisonQueueTopic != "" {
		return c.PoisonQueueTopic
	}
	return c.ConsumerGroup + ".failed"
}

func buildSaramaConfig(cfg Config) (*sarama.Config, error) {
	sc := sarama.NewConfig()
	sc.Version = sarama.V2_8_0_0

	if cfg.SASL.Enable {
		sc.Net.SASL.Enable = true
		sc.Net.SASL.User = cfg.SASL.Username
		sc.Net.SASL.Password = cfg.SASL.Password
		switch cfg.SASL.Mechanism {
		case "SCRAM-SHA-256":
			sc.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &scramClient{HashGeneratorFcn: scram.SHA256}
			}
			sc.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			sc.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &scramClient{HashGeneratorFcn: scram.SHA512}
			}
			sc.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		default:
			sc.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	if cfg.TLS.Enable {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify, //nolint:gosec // InsecureSkipVerify is user-controlled via config
		}
		if cfg.TLS.CACert != "" {
			pem, err := os.ReadFile(cfg.TLS.CACert)
			if err != nil {
				return nil, Domain.Wrapf(err, "read ca cert %s", cfg.TLS.CACert)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(pem)
			tlsCfg.RootCAs = pool
		}
		if cfg.TLS.ClientCert != "" && cfg.TLS.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.ClientCert, cfg.TLS.ClientKey)
			if err != nil {
				return nil, Domain.Wrapf(err, "load client cert/key")
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
		sc.Net.TLS.Enable = true
		sc.Net.TLS.Config = tlsCfg
	}

	return sc, nil
}

// scramClient implements sarama.SCRAMClient via xdg-go/scram.
type scramClient struct {
	HashGeneratorFcn   scram.HashGeneratorFcn
	client             *scram.Client
	clientConversation *scram.ClientConversation
}

func (x *scramClient) Begin(userName, password, authzID string) (err error) {
	x.client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return Domain.Wrap(err, "scram begin")
	}
	x.clientConversation = x.client.NewConversation()
	return nil
}

func (x *scramClient) Step(challenge string) (string, error) {
	response, err := x.clientConversation.Step(challenge)
	return response, Domain.Wrap(err, "scram step")
}

func (x *scramClient) Done() bool { return x.clientConversation.Done() }
