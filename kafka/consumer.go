package kafka

import (
    "context" 
    "github.com/IBM/sarama"
    "log"
)

type Consumer struct {
    consumerGroup sarama.ConsumerGroup
    topics        []string
    handler       OrderHandler
}


func NewConsumer(brokers []string, groupID string, topics []string, 
                 config *sarama.Config, handler OrderHandler) (*Consumer, error) {
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


func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
    return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
    return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for message := range claim.Messages() {
        err := c.handler.Handle(session.Context(), message)
        if err == nil {
            session.MarkMessage(message, "")
        } else {
            log.Printf("Failed to process message: %v", err)
        }
    }
    return nil
}

func (c *Consumer) Start(ctx context.Context) error {
    for {
        if ctx.Err() != nil {
            return nil
        }
        if err := c.consumerGroup.Consume(ctx, c.topics, c); err != nil {
            if err == sarama.ErrClosedConsumerGroup {
                return nil
            }
        }
    }
}

func (c *Consumer) Close() error {
    return c.consumerGroup.Close()
}