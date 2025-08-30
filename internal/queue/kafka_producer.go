package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/pkg/config"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	logger   *logrus.Entry
}

func NewKafkaProducer(cfg *config.KafkaConfig) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = cfg.RetryAttempts
	saramaConfig.Producer.Retry.Backoff = time.Millisecond * 250
	saramaConfig.Producer.Partitioner = sarama.NewRandomPartitioner
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Producer.Flush.Frequency = time.Millisecond * 500

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger := logrus.WithField("component", "kafka_producer")
	logger.Info("Kafka producer created successfully")

	return &KafkaProducer{
		producer: producer,
		topic:    cfg.OrderTopic,
		logger:   logger,
	}, nil
}

func (p *KafkaProducer) PublishEvent(ctx context.Context, event *models.Event) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		p.logger.WithError(err).Error("Failed to marshal event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	key := event.ID.String()
	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(eventData),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event_type"),
				Value: []byte(event.Type),
			},
			{
				Key:   []byte("event_id"),
				Value: []byte(event.ID.String()),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(event.Timestamp.Format(time.RFC3339)),
			},
		},
		Timestamp: event.Timestamp,
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err,
		}).Error("Failed to publish event to Kafka")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"partition":  partition,
		"offset":     offset,
	}).Info("Event published successfully")

	return nil
}

func (p *KafkaProducer) Close() error {
	if p.producer != nil {
		if err := p.producer.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to close Kafka producer")
			return fmt.Errorf("failed to close producer: %w", err)
		}
		p.logger.Info("Kafka producer closed successfully")
	}
	return nil
}