package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/queue"
	"order-processing-microservice/internal/repository"
	"order-processing-microservice/internal/services"
	"order-processing-microservice/pkg/config"
	"order-processing-microservice/pkg/database"
	"order-processing-microservice/pkg/logger"
)

func main() {
	configFile := "configs/local.env"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		// Fallback to environment variables only
		logrus.Warnf("Config file not found, using environment variables: %v", err)
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:         getEnv("DATABASE_HOST", "localhost"),
				Port:         getEnvInt("DATABASE_PORT", 5432),
				Username:     getEnv("DATABASE_USERNAME", "postgres"),
				Password:     getEnv("DATABASE_PASSWORD", "postgres"),
				Database:     getEnv("DATABASE_DATABASE", "orders"),
				SSLMode:      getEnv("DATABASE_SSL_MODE", "disable"),
				MaxOpenConns: getEnvInt("DATABASE_MAX_OPEN_CONNS", 25),
				MaxIdleConns: getEnvInt("DATABASE_MAX_IDLE_CONNS", 5),
			},
			Kafka: config.KafkaConfig{
				Brokers:          []string{getEnv("KAFKA_BROKERS", "kafka:9092")},
				GroupID:          getEnv("KAFKA_GROUP_ID", "order-processing-group"),
				OrderTopic:       getEnv("KAFKA_ORDER_TOPIC", "order-events"),
				RetryAttempts:    getEnvInt("KAFKA_RETRY_ATTEMPTS", 3),
				SessionTimeout:   getEnvInt("KAFKA_SESSION_TIMEOUT", 30000),
				CommitInterval:   getEnvInt("KAFKA_COMMIT_INTERVAL", 1000),
				EnableAutoCommit: getEnvBool("KAFKA_ENABLE_AUTO_COMMIT", true),
			},
			Logger: config.LoggerConfig{
				Level:  getEnv("LOGGER_LEVEL", "info"),
				Format: getEnv("LOGGER_FORMAT", "json"),
			},
		}
	}

	logger.Init(&cfg.Logger)

	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	producer, err := queue.NewKafkaProducer(&cfg.Kafka)
	if err != nil {
		logrus.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	consumer, err := queue.NewKafkaConsumer(&cfg.Kafka)
	if err != nil {
		logrus.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	orderRepo := repository.NewPostgresOrderRepository(db.GetDB())
	orderProcessor := services.NewOrderProcessor(orderRepo, producer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := consumer.Subscribe(ctx, orderProcessor); err != nil {
		logrus.Fatalf("Failed to subscribe to Kafka topics: %v", err)
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := orderProcessor.ProcessPendingOrders(ctx); err != nil {
					logrus.WithError(err).Error("Failed to process pending orders")
				}
			}
		}
	}()

	logrus.Info("Order processing consumer started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down consumer...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		consumer.Close()
		close(done)
	}()

	select {
	case <-done:
		logrus.Info("Consumer stopped gracefully")
	case <-shutdownCtx.Done():
		logrus.Error("Consumer shutdown timeout exceeded")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}