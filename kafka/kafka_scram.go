package kafka

import (
	"LiteAdmin/config"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

// SCRAM认证
func NewSaramaConfigWithSCRAM(cfg *config.KafkaConfig, mechanism string) (*sarama.Config, error) {
    config := sarama.NewConfig()
    config.Version = sarama.V2_8_0_0
    
    // 基础配置
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Return.Successes = true
    config.Consumer.Return.Errors = true
    
    // SCRAM认证配置
    config.Net.SASL.Enable = true
    config.Net.SASL.User = cfg.Username
    config.Net.SASL.Password = cfg.Password
    config.Net.SASL.Handshake = true
    
    // 选择SCRAM机制
    switch mechanism {
    case "SCRAM-SHA-256":
        config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
        config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
            return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
        }
    case "SCRAM-SHA-512":
        config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
        config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
            return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
        }
    default:
        // 默认使用PLAIN
        config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
    }
    
    // TLS配置
    if cfg.UseTLS {
        tlsConfig, err := createTLSConfig(cfg.CertFile, cfg.KeyFile, cfg.CAFile)
        if err != nil {
            return nil, err
        }
        config.Net.TLS.Enable = true
        config.Net.TLS.Config = tlsConfig
    }
    
    return config, nil
}

// SCRAM客户端实现
var (
    SHA256 scram.HashGeneratorFcn = sha256HashGenerator()
    SHA512 scram.HashGeneratorFcn = sha512HashGenerator()
)

func sha256HashGenerator() scram.HashGeneratorFcn {
    return scram.SHA256
}

func sha512HashGenerator() scram.HashGeneratorFcn {
    return scram.SHA512
}

type XDGSCRAMClient struct {
    *scram.Client
    *scram.ClientConversation
    scram.HashGeneratorFcn
}

func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
    x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
    if err != nil {
        return err
    }
    x.ClientConversation = x.Client.NewConversation()
    return nil
}

func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
    response, err = x.ClientConversation.Step(challenge)
    return
}

func (x *XDGSCRAMClient) Done() bool {
    return x.ClientConversation.Done()
}