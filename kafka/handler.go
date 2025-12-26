package kafka

import (
    "context"
    "encoding/json"
    "log"
    
    "github.com/IBM/sarama"
)

type OrderMessage struct {
    OrderID   string  `json:"order_id"`
    UserID    string  `json:"user_id"`
    Amount    float64 `json:"amount"`
    Timestamp int64   `json:"timestamp"`
}

type OrderHandler struct {
}

func NewOrderHandler() *OrderHandler {
    return &OrderHandler{}
}

func (h *OrderHandler) Handle(ctx context.Context, message *sarama.ConsumerMessage) error {
    var order OrderMessage
    
    if err := json.Unmarshal(message.Value, &order); err != nil {
        log.Printf("Failed to unmarshal message: %v", err)
        return err
    }
    
    log.Printf("Processing order: %+v", order)
    
    // TODO 业务处理逻辑
    
    
    return nil
}