package kafka

import (
    "context" 
    "log"
    "sync"
    
    "github.com/IBM/sarama"
)

type Consumer struct {
    consumerGroup sarama.ConsumerGroup
    topics        []string
    handler       MessageHandler
}

type MessageHandler interface {
    Handle(ctx context.Context, message *sarama.ConsumerMessage) error
}

func NewConsumer(brokers []string, groupID string, topics []string, 
                 config *sarama.Config, handler MessageHandler) (*Consumer, error) {
    consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
    if err != nil {
        return nil, err
    }
    
    return &Consumer{
        consumerGroup: consumerGroup,
        topics:        topics,
        handler:       handler,
    }, nil
}

func (c *Consumer) Start(ctx context.Context) error {
    wg := &sync.WaitGroup{}
    wg.Add(1)
    
    go func() {
        defer wg.Done()
        for {
            if err := c.consumerGroup.Consume(ctx, c.topics, c); err != nil {
                log.Printf("Error from consumer: %v", err)
            }
            
            if ctx.Err() != nil {
                return
            }
        }
    }()
    
    wg.Wait()
    return nil
}

// 实现 ConsumerGroupHandler 接口
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
    return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
    return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, 
                                claim sarama.ConsumerGroupClaim) error {
    for message := range claim.Messages() {
        log.Printf("Received message: topic=%s, partition=%d, offset=%d",
            message.Topic, message.Partition, message.Offset)
        
        // 调用业务处理器
        if err := c.handler.Handle(session.Context(), message); err != nil {
            log.Printf("Error handling message: %v", err)
            // 根据业务需求决定是否继续或重试
        }
        
        // 标记消息已处理
        session.MarkMessage(message, "")
    }
    
    return nil
}

func (c *Consumer) Close() error {
    return c.consumerGroup.Close()
}