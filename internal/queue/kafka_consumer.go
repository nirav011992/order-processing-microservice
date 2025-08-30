package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/pkg/config"
)

type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	topic         string
	groupID       string
	handler       EventHandler
	logger        *logrus.Entry
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

type consumerGroupHandler struct {
	handler EventHandler
	logger  *logrus.Entry
}

func NewKafkaConsumer(cfg *config.KafkaConfig) (*KafkaConsumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Group.Session.Timeout = time.Duration(cfg.SessionTimeout) * time.Millisecond
	saramaConfig.Consumer.Group.Heartbeat.Interval = time.Second * 3
	saramaConfig.Consumer.MaxProcessingTime = time.Second * 30
	saramaConfig.Consumer.Return.Errors = true

	if cfg.EnableAutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = time.Duration(cfg.CommitInterval) * time.Millisecond
	}

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}

	logger := logrus.WithFields(logrus.Fields{
		"component": "kafka_consumer",
		"group_id":  cfg.GroupID,
		"topic":     cfg.OrderTopic,
	})
	logger.Info("Kafka consumer created successfully")

	return &KafkaConsumer{
		consumerGroup: consumerGroup,
		topic:         cfg.OrderTopic,
		groupID:       cfg.GroupID,
		logger:        logger,
	}, nil
}

func (c *KafkaConsumer) Subscribe(ctx context.Context, handler EventHandler) error {
	c.handler = handler

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	groupHandler := &consumerGroupHandler{
		handler: handler,
		logger:  c.logger,
	}

	c.wg.Add(2)

	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := c.consumerGroup.Consume(ctx, []string{c.topic}, groupHandler); err != nil {
					c.logger.WithError(err).Error("Error consuming messages")
					time.Sleep(time.Second)
				}
			}
		}
	}()

	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-c.consumerGroup.Errors():
				if err != nil {
					c.logger.WithError(err).Error("Consumer group error")
				}
			}
		}
	}()

	c.logger.Info("Started consuming messages")
	return nil
}

func (c *KafkaConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	c.wg.Wait()

	if c.consumerGroup != nil {
		if err := c.consumerGroup.Close(); err != nil {
			c.logger.WithError(err).Error("Failed to close consumer group")
			return fmt.Errorf("failed to close consumer group: %w", err)
		}
		c.logger.Info("Kafka consumer closed successfully")
	}
	return nil
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Consumer group session started")
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Consumer group session ended")
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			if err := h.processMessage(session.Context(), message); err != nil {
				h.logger.WithFields(logrus.Fields{
					"partition": message.Partition,
					"offset":    message.Offset,
					"error":     err,
				}).Error("Failed to process message")
				continue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *consumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event models.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		h.logger.WithError(err).Error("Failed to unmarshal event")
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"partition":  message.Partition,
		"offset":     message.Offset,
	}).Info("Processing event")

	if err := h.handler.HandleEvent(ctx, &event); err != nil {
		h.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"error":      err,
		}).Error("Handler failed to process event")
		return fmt.Errorf("handler failed to process event: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
	}).Info("Event processed successfully")

	return nil
}