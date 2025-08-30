package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/handlers"
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
			Server: config.ServerConfig{
				Host:         getEnv("SERVER_HOST", "localhost"),
				Port:         getEnvInt("SERVER_PORT", 8080),
				ReadTimeout:  getEnvInt("SERVER_READ_TIMEOUT", 10),
				WriteTimeout: getEnvInt("SERVER_WRITE_TIMEOUT", 10),
			},
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

	if err := db.CreateTables(); err != nil {
		logrus.Fatalf("Failed to create database tables: %v", err)
	}

	producer, err := queue.NewKafkaProducer(&cfg.Kafka)
	if err != nil {
		logrus.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	orderRepo := repository.NewPostgresOrderRepository(db.GetDB())
	orderService := services.NewOrderService(orderRepo, producer)
	producerHandlers := handlers.NewProducerHandlers(orderService)

	r := gin.New()
	r.Use(handlers.LoggerMiddleware())
	r.Use(handlers.CORSMiddleware())
	r.Use(handlers.SecurityHeadersMiddleware())
	r.Use(handlers.RequestIDMiddleware())
	r.Use(gin.Recovery())

	producerHandlers.RegisterRoutes(r)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		logrus.Infof("Producer API server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down Producer API server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Producer API server forced to shutdown: %v", err)
	}

	logrus.Info("Producer API server stopped")
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