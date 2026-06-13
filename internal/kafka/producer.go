package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/rs/zerolog/log"
)

// Producer handles serialising and writing events to Kafka.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer constructs a Producer connected to the given brokers.
func NewProducer(brokers []string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		// If Topic is not specified on the writer, it can be set per Message.
		Async:        false, 
	}

	log.Info().Interface("brokers", brokers).Msg("Kafka Producer initialized")
	return &Producer{writer: writer}
}

// PublishEvent serialises the payload to JSON and publishes it to the specified topic.
func (p *Producer) PublishEvent(ctx context.Context, topic string, eventType string, payload interface{}) error {
	// However, to make consumers' life easy, we can also marshall the whole wrapper Event struct.
	// Let's marshall the wrapper:
	eventWrapper := struct {
		Type      string      `json:"type"`
		Timestamp time.Time   `json:"timestamp"`
		Payload   interface{} `json:"payload"`
	}{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	msgBytes, err := json.Marshal(eventWrapper)
	if err != nil {
		return fmt.Errorf("publish event: marshal wrapper: %w", err)
	}

	log.Debug().
		Str("topic", topic).
		Str("event_type", eventType).
		Msg("Publishing event to Kafka...")

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano())),
		Value: msgBytes,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(eventType)},
		},
	})
	if err != nil {
		return fmt.Errorf("publish event: write message: %w", err)
	}

	return nil
}

// Close shuts down the Kafka writer connection.
func (p *Producer) Close() error {
	log.Info().Msg("Closing Kafka Producer...")
	return p.writer.Close()
}
