package main

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func main() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"kafka:9092"},
		Topic:     "stats",
		GroupID:   "stats-consumer-group",
		Partition: 0,
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})

	log.Println("Stats service started. Listening for messages...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Could not read message: %v", err)
			continue
		}
		log.Printf("Received: %s", string(msg.Value))
	}
}
