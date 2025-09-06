// server/kafka_consumer.go
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type ConsumerRole string

const (
	RolePersister   ConsumerRole = "persister"
	RoleBroadcaster ConsumerRole = "broadcaster"
)

type KafkaConsumer struct {
	reader *kafka.Reader
	db     *pgxpool.Pool
	hub    *Hub
	role   ConsumerRole
}

func NewKafkaConsumer(brokers []string, topic string, groupID string, db *pgxpool.Pool, hub *Hub, role ConsumerRole) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &KafkaConsumer{reader: reader, db: db, hub: hub, role: role}
}

func (c *KafkaConsumer) Run(ctx context.Context) {
	log.Printf("Kafka consumer starting with role: %s", c.role)
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("could not fetch message (role: %s): %v", c.role, err)
			break
		}

		switch c.role {
		case RoleBroadcaster:
			var msg DocumentUpdateMessage
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("could not unmarshal message for broadcast: %v", err)
				continue
			}
			c.hub.broadcast(msg.DocumentID, m.Value)

		case RolePersister:
			var msg DocumentUpdateMessage
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("could not unmarshal message for persistence: %v", err)
				continue
			}
			query := `UPDATE documents SET content = $1, updated_at = NOW() WHERE id = $2`
			_, err = c.db.Exec(context.Background(), query, msg.Content, msg.DocumentID)
			if err != nil {
				log.Printf("Failed to update document in DB: %v", err)
				continue
			}
			log.Printf("Successfully persisted update for document %s", msg.DocumentID)
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("failed to commit messages: %v", err)
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
