package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type DocumentUpdateMessage struct {
	DocumentID string `json:"document_id"`
	ClientID   string `json:"client_id"`
	Content    string `json:"content"`
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaProducer{
		writer,
	}
}

func (p *KafkaProducer) ProduceMessage(ctx context.Context, msg DocumentUpdateMessage) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("could not marshal message: %v", err)
		return err
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.DocumentID), // DocumentID as the key to ensure messages for the same document go to the same partition
		Value: msgBytes,
	}); err != nil {
		log.Printf("could not write message: %v", err)
		return err
	}

	log.Printf("Message produced for document %s", msg.DocumentID)
	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
