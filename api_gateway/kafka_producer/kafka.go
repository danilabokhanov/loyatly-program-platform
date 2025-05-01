package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

var writer *kafka.Writer

func initKafka(brokerAddress string, topic string) {
	writer = &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
}

func SendStat(eventType string, userID string, objectID string) {
	if writer == nil {
		initKafka("kafka:9092", "stats")
	}

	message := map[string]interface{}{
		"event_type": eventType,
		"user_id":    userID,
		"object_id":  objectID,
		"timestamp":  time.Now().Unix(),
	}

	bytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal kafka message: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = writer.WriteMessages(
		ctx,
		kafka.Message{
			Key:   []byte(userID),
			Value: bytes,
		},
	)
	if err != nil {
		log.Printf("Failed to send message to Kafka: %v\n", err)
	}
}
