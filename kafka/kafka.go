package kafka

import (
	"LiteAdmin/config"
	"crypto/tls"
	"crypto/x509"
	"os"
	"github.com/IBM/sarama"
)

func NewSaramaConfig(cfg *config.KafkaConfig) (*sarama.Config, error) {
    config := sarama.NewConfig()
    config.Version = sarama.V2_8_0_0
    
    // 生产者配置
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Retry.Max = 3
    config.Producer.Return.Successes = true
    config.Producer.Partitioner = sarama.NewRandomPartitioner
    config.Producer.Interceptors=[]sarama.ProducerInterceptor{NewOrderInterceptor()}
    
    // 消费者配置
    config.Consumer.Return.Errors = true
    config.Consumer.Offsets.Initial = sarama.OffsetNewest
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    
    // 认证配置
    
    // 1. SASL/PLAIN 认证
    if cfg.Username != "" && cfg.Password != "" {
        config.Net.SASL.Enable = true
        config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
        config.Net.SASL.User = cfg.Username
        config.Net.SASL.Password = cfg.Password
        config.Net.SASL.Handshake = true
    }
    
    // 2. TLS 配置
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

// 创建TLS配置
func createTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
    tlsConfig := &tls.Config{}
    
    // 加载CA证书
    if caFile != "" {
        caCert, err := os.ReadFile(caFile)
        if err != nil {
            return nil, err
        }
        
        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)
        tlsConfig.RootCAs = caCertPool
    }
    
    // 加载客户端证书
    if certFile != "" && keyFile != "" {
        cert, err := tls.LoadX509KeyPair(certFile, keyFile)
        if err != nil {
            return nil, err
        }
        tlsConfig.Certificates = []tls.Certificate{cert}
    }
    
    // 生产环境建议设置为true
    tlsConfig.InsecureSkipVerify = false
    
    return tlsConfig, nil
}