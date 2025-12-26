package kafka

import (
	"log"
	"github.com/IBM/sarama"
)
type OrderInterceptor struct{

}

func (i *OrderInterceptor) OnSend(msg *sarama.ProducerMessage) {
    log.Printf("拦截到准备发送的消息，Topic: %s", msg.Topic)
    msg.Headers = append(msg.Headers, sarama.RecordHeader{
        Key:   []byte("intercepted-by"),
        Value: []byte("OrderInterceptor"),
    })
}

func NewOrderInterceptor() *OrderInterceptor {
    return &OrderInterceptor{
    }
}