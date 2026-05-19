package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog"
)

// Producer wraps the confluent-kafka-go producer with delivery tracking.
// We use synchronous delivery confirmation (wait for ack) because the gateway
// must not return ACCEPTED to the caller until Kafka has acknowledged the write.
// Returning ACCEPTED before ack would make the guarantee hollow.
type Producer struct {
	client  *kafka.Producer
	timeout time.Duration
	log     zerolog.Logger
}

func NewProducer(bootstrapServers string, timeout time.Duration, log zerolog.Logger) (*Producer, error) {
	client, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":   bootstrapServers,
		"acks":                "all",
		"enable.idempotence":  true,
		"compression.type":    "snappy",
		// Limit in-flight requests to 1 per connection when idempotence is on.
		// This is required by the Kafka protocol for exactly-once producer semantics.
		"max.in.flight.requests.per.connection": 1,
		"retries":             3,
		"retry.backoff.ms":    100,
	})
	if err != nil {
		return nil, fmt.Errorf("creating kafka producer: %w", err)
	}

	return &Producer{
		client:  client,
		timeout: timeout,
		log:     log.With().Str("component", "kafka_producer").Logger(),
	}, nil
}

// Produce writes a message to Kafka and waits for broker acknowledgment.
// Returns the Kafka message offset as a string message ID on success.
// The context deadline is respected: if the context is cancelled before
// delivery, the message may or may not have been written to Kafka.
// Callers should treat context cancellation as an unknown outcome.
func (p *Producer) Produce(ctx context.Context, topic, key string, payload []byte) (string, error) {
	deliveryChan := make(chan kafka.Event, 1)

	err := p.client.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(key),
		Value: payload,
	}, deliveryChan)
	if err != nil {
		return "", fmt.Errorf("enqueuing message: %w", err)
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case event := <-deliveryChan:
		msg, ok := event.(*kafka.Message)
		if !ok {
			return "", fmt.Errorf("unexpected delivery event type: %T", event)
		}
		if msg.TopicPartition.Error != nil {
			return "", fmt.Errorf("kafka delivery failed: %w", msg.TopicPartition.Error)
		}
		msgID := fmt.Sprintf("%s:%d:%d", topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
		return msgID, nil
	}
}

func (p *Producer) Close() {
	// Flush waits for all enqueued messages to be delivered before closing.
	// The 5s timeout prevents a hung producer from blocking graceful shutdown indefinitely.
	remaining := p.client.Flush(5000)
	if remaining > 0 {
		p.log.Warn().Int("unflushed", remaining).Msg("kafka producer closed with unflushed messages")
	}
	p.client.Close()
}
