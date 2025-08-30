# Deployment Guide

This document provides comprehensive deployment instructions for the Order Processing Microservice across different environments.

## Table of Contents

1. [Docker Compose Deployment](#docker-compose-deployment)
2. [Environment Configuration](#environment-configuration)
3. [Production Deployment](#production-deployment)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Monitoring and Observability](#monitoring-and-observability)
6. [Troubleshooting](#troubleshooting)

## Docker Compose Deployment

The recommended way to deploy the Order Processing Microservice for development and testing is using Docker Compose.

### Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- At least 2GB RAM available
- Ports 5432, 8080, 9080, 9092, and 29092 available

### Quick Start

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd order-processing-microservice
   ```

2. **Start all services:**
   ```bash
   docker-compose up -d
   ```

3. **Verify deployment:**
   ```bash
   # Check service health
   curl http://localhost:8080/health
   curl http://localhost:9080/health
   
   # View service logs
   docker-compose logs -f
   ```

4. **Stop services:**
   ```bash
   docker-compose down
   ```

### Service Architecture

The Docker Compose setup includes the following services:

| Service | Port | Description |
|---------|------|-------------|
| postgres | 5432 | PostgreSQL database |
| zookeeper | 2181 | Apache Zookeeper for Kafka |
| kafka | 9092, 29092 | Apache Kafka message broker |
| producer-api | 8080 | Order creation and management API |
| consumer | - | Background order processing service |
| status-api | 9080 | Order monitoring and statistics API |

### Docker Networks

All services run on a custom bridge network `order-processing-network` which enables:
- Service discovery by name (e.g., `kafka:9092`)
- Network isolation from other Docker applications
- Secure inter-service communication

### Volume Management

```bash
# List volumes
docker volume ls | grep order

# Backup database
docker exec order-postgres pg_dump -U postgres orders_db > backup.sql

# Restore database (with services stopped)
cat backup.sql | docker exec -i order-postgres psql -U postgres -d orders_db
```

### Health Checks

All services include health checks:

```bash
# Check service health status
docker-compose ps

# View health check logs
docker inspect order-kafka --format='{{json .State.Health}}'
```

## Environment Configuration

### Configuration Files

The system supports multiple environment configurations:

- `configs/local.env` - Local development (default)
- `configs/staging.env` - Staging environment
- `configs/production.env` - Production environment

### Environment Variables

#### Database Configuration

```env
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USERNAME=postgres
DATABASE_PASSWORD=secure_password
DATABASE_DATABASE=orders_db
DATABASE_SSL_MODE=require
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
```

#### Kafka Configuration

```env
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
KAFKA_GROUP_ID=order-processing-group
KAFKA_ORDER_TOPIC=order-events
KAFKA_RETRY_ATTEMPTS=3
KAFKA_SESSION_TIMEOUT=30000
KAFKA_COMMIT_INTERVAL=1000
KAFKA_ENABLE_AUTO_COMMIT=true
```

#### Server Configuration

```env
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30
```

#### Logging Configuration

```env
LOGGER_LEVEL=info
LOGGER_FORMAT=json
```

### Docker Compose Override

For local customization, create `docker-compose.override.yml`:

```yaml
version: '3.8'

services:
  producer-api:
    ports:
      - "8081:8080"  # Custom port mapping
    environment:
      LOGGER_LEVEL: debug
    volumes:
      - ./logs:/var/log/app

  postgres:
    environment:
      POSTGRES_PASSWORD: custom_password
    volumes:
      - ./custom-init.sql:/docker-entrypoint-initdb.d/init.sql
```

## Production Deployment

### Production Considerations

1. **Security:**
   - Use strong passwords and secrets
   - Enable TLS/SSL for all communications
   - Configure proper network policies
   - Regular security updates

2. **High Availability:**
   - Multiple replicas for each service
   - Load balancers for API services
   - Kafka cluster with multiple brokers
   - Database replication

3. **Performance:**
   - Resource limits and requests
   - Connection pooling
   - Caching strategies
   - Database indexing

### Production Docker Compose

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  producer-api:
    image: order-processing/producer:${VERSION}
    deploy:
      replicas: 3
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
    environment:
      DATABASE_PASSWORD_FILE: /run/secrets/db_password
      LOGGER_LEVEL: warn
    secrets:
      - db_password
    networks:
      - app-network
      - db-network

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
    secrets:
      - db_password
    networks:
      - db-network

secrets:
  db_password:
    file: ./secrets/db_password.txt

networks:
  app-network:
    driver: overlay
  db-network:
    driver: overlay
```

### SSL/TLS Configuration

1. **Generate certificates:**
   ```bash
   # Create CA
   openssl genrsa -out ca-key.pem 4096
   openssl req -new -x509 -days 365 -key ca-key.pem -out ca.pem
   
   # Create server certificate
   openssl genrsa -out server-key.pem 4096
   openssl req -new -key server-key.pem -out server.csr
   openssl x509 -req -days 365 -in server.csr -CA ca.pem -CAkey ca-key.pem -out server.pem
   ```

2. **Configure services:**
   ```yaml
   services:
     producer-api:
       volumes:
         - ./certs/server.pem:/etc/ssl/certs/server.pem:ro
         - ./certs/server-key.pem:/etc/ssl/private/server-key.pem:ro
       environment:
         TLS_CERT_FILE: /etc/ssl/certs/server.pem
         TLS_KEY_FILE: /etc/ssl/private/server-key.pem
   ```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster 1.24+
- kubectl configured
- Helm 3.0+ (optional)

### Namespace Setup

```bash
# Create namespace
kubectl create namespace order-processing

# Set default namespace
kubectl config set-context --current --namespace=order-processing
```

### ConfigMap and Secrets

```bash
# Create configuration
kubectl create configmap order-config \
  --from-env-file=configs/production.env

# Create database secret
kubectl create secret generic db-secret \
  --from-literal=password=secure_db_password

# Create TLS secret
kubectl create secret tls order-tls \
  --cert=server.pem \
  --key=server-key.pem
```

### Database Deployment

```yaml
# postgres-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_DB
          value: orders_db
        - name: POSTGRES_USER
          value: postgres
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          limits:
            memory: "1Gi"
            cpu: "500m"
          requests:
            memory: "512Mi"
            cpu: "250m"
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

### Application Deployment

```yaml
# producer-api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: producer-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: producer-api
  template:
    metadata:
      labels:
        app: producer-api
    spec:
      containers:
      - name: producer-api
        image: order-processing/producer:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: order-config
        env:
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
          requests:
            memory: "256Mi"
            cpu: "250m"
---
apiVersion: v1
kind: Service
metadata:
  name: producer-api-service
spec:
  selector:
    app: producer-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Ingress Configuration

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: order-processing-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.orders.example.com
    secretName: order-tls
  rules:
  - host: api.orders.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: producer-api-service
            port:
              number: 80
```

### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: producer-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: producer-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Deployment Commands

```bash
# Apply all configurations
kubectl apply -f k8s/

# Check deployment status
kubectl get pods
kubectl get services
kubectl get ingress

# View logs
kubectl logs -f deployment/producer-api
kubectl logs -f deployment/consumer

# Scale services
kubectl scale deployment producer-api --replicas=5
```

## Monitoring and Observability

### Health Check Endpoints

```bash
# Check service health
curl http://localhost:8080/health
curl http://localhost:9080/health

# Get metrics
curl http://localhost:9080/api/v1/status/metrics
```

### Log Aggregation

For production, configure centralized logging:

```yaml
# docker-compose with log driver
services:
  producer-api:
    logging:
      driver: "fluentd"
      options:
        fluentd-address: logging-server:24224
        tag: "order.producer"
```

### Prometheus Metrics

Add Prometheus metrics endpoint (future enhancement):

```go
// Example metrics endpoint
http.Handle("/metrics", promhttp.Handler())
```

### Jaeger Tracing

Configure distributed tracing:

```yaml
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      COLLECTOR_ZIPKIN_HTTP_PORT: 9411
```

## Troubleshooting

### Common Issues

1. **Services not starting:**
   ```bash
   # Check logs
   docker-compose logs service-name
   
   # Check health
   docker-compose ps
   
   # Restart specific service
   docker-compose restart service-name
   ```

2. **Kafka connection issues:**
   ```bash
   # Check Kafka topics
   docker exec order-kafka kafka-topics --list --bootstrap-server localhost:9092
   
   # View consumer groups
   docker exec order-kafka kafka-consumer-groups --list --bootstrap-server localhost:9092
   ```

3. **Database connection problems:**
   ```bash
   # Connect to database
   docker exec -it order-postgres psql -U postgres -d orders_db
   
   # Check database logs
   docker-compose logs postgres
   ```

4. **Port conflicts:**
   ```bash
   # Check port usage
   netstat -tulpn | grep :8080
   
   # Use different ports
   docker-compose -f docker-compose.yml -f docker-compose.override.yml up
   ```

### Performance Tuning

1. **Database optimization:**
   - Increase connection pool size
   - Add database indexes
   - Configure database caching

2. **Kafka optimization:**
   - Increase partition count
   - Configure batch size
   - Adjust consumer concurrency

3. **Application tuning:**
   - Increase worker threads
   - Configure HTTP timeouts
   - Enable connection pooling

### Backup and Recovery

1. **Database backup:**
   ```bash
   # Create backup
   docker exec order-postgres pg_dump -U postgres orders_db > backup-$(date +%Y%m%d).sql
   
   # Restore backup
   cat backup-20250830.sql | docker exec -i order-postgres psql -U postgres -d orders_db
   ```

2. **Configuration backup:**
   ```bash
   # Backup configurations
   tar -czf config-backup-$(date +%Y%m%d).tar.gz configs/ docker-compose.yml
   ```

### Monitoring Commands

```bash
# Check resource usage
docker stats

# View service logs in real-time
docker-compose logs -f --tail=100

# Check network connectivity
docker exec producer-api ping postgres
docker exec producer-api telnet kafka 9092

# Verify API endpoints
curl -v http://localhost:8080/health
curl -s http://localhost:9080/api/v1/status/stats | jq
```

## Security Best Practices

1. **Secrets Management:**
   - Use Docker secrets or Kubernetes secrets
   - Never commit secrets to version control
   - Rotate secrets regularly

2. **Network Security:**
   - Use custom Docker networks
   - Implement network policies in Kubernetes
   - Enable TLS for all communications

3. **Image Security:**
   - Use minimal base images
   - Scan images for vulnerabilities
   - Keep dependencies updated

4. **Access Control:**
   - Implement proper RBAC
   - Use service accounts
   - Limit container privileges

## Scaling Strategies

1. **Horizontal Scaling:**
   - Scale API services based on CPU/memory
   - Use load balancers
   - Implement circuit breakers

2. **Database Scaling:**
   - Read replicas for queries
   - Connection pooling
   - Database sharding (if needed)

3. **Message Queue Scaling:**
   - Increase Kafka partitions
   - Multiple consumer instances
   - Consumer group optimization