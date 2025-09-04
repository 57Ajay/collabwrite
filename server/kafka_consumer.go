package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
	db     *pgxpool.Pool
}

func NewKafkaConsumer(brokers []string, topic string, groupId string, db *pgxpool.Pool) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupId,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 1MB
	})

	return &KafkaConsumer{reader, db}
}

func (c *KafkaConsumer) Run(ctx context.Context) {
	log.Println("Kafka consumer is running...")

	for {
		// FetchMessage blocks until a new message is available
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("could not fetch message: %v", err)
			break
		}

		var msg DocumentUpdateMessage

		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("could not unmarshal message: %v", err)
			continue
		}

		log.Printf("Message consumed for document %s", msg.DocumentID)

		query := `UPDATE documents SET content = $1, updated_at = NOW() WHERE id = $2`
		_, err = c.db.Exec(context.Background(), query, msg.Content, msg.DocumentID)
		if err != nil {
			log.Printf("Failed to update document in DB: %v", err)
			// TODO: handle this error more gracefully (e.g., retry or send to a dead-letter queue)
			continue
		}

		log.Printf("Successfully persisted update for document %s", msg.DocumentID)

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("failed to commit messages: %v", err)
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
