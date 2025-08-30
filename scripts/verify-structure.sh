#!/bin/bash

echo "=== Order Processing Microservice Structure Verification ==="
echo ""

# Check Go module
if [ -f "go.mod" ]; then
    echo "✅ go.mod found"
    echo "   Module: $(grep '^module' go.mod)"
else
    echo "❌ go.mod missing"
fi

# Check main applications
echo ""
echo "📦 Main Applications:"
for app in producer consumer status-api; do
    if [ -f "cmd/$app/main.go" ]; then
        echo "   ✅ cmd/$app/main.go"
    else
        echo "   ❌ cmd/$app/main.go missing"
    fi
done

# Check core packages
echo ""
echo "📋 Core Packages:"
packages=(
    "internal/models/order.go"
    "internal/models/events.go"
    "internal/handlers/producer_handlers.go"
    "internal/handlers/status_handlers.go"
    "internal/handlers/middleware.go"
    "internal/queue/kafka_producer.go"
    "internal/queue/kafka_consumer.go"
    "internal/repository/order_repository.go"
    "internal/services/order_service.go"
    "internal/services/order_processor.go"
    "pkg/config/config.go"
    "pkg/database/postgres.go"
    "pkg/logger/logger.go"
    "pkg/utils/response.go"
)

for package in "${packages[@]}"; do
    if [ -f "$package" ]; then
        echo "   ✅ $package"
    else
        echo "   ❌ $package missing"
    fi
done

# Check configuration files
echo ""
echo "⚙️  Configuration Files:"
configs=(
    "configs/local.env"
    "configs/staging.env"
    "configs/production.env"
)

for config in "${configs[@]}"; do
    if [ -f "$config" ]; then
        echo "   ✅ $config"
    else
        echo "   ❌ $config missing"
    fi
done

# Check Docker files
echo ""
echo "🐳 Docker Files:"
dockerfiles=(
    "Dockerfile.producer"
    "Dockerfile.consumer"
    "Dockerfile.status"
    "docker-compose.yml"
)

for dockerfile in "${dockerfiles[@]}"; do
    if [ -f "$dockerfile" ]; then
        echo "   ✅ $dockerfile"
    else
        echo "   ❌ $dockerfile missing"
    fi
done

# Check build tools
echo ""
echo "🔨 Build Tools:"
if [ -f "Makefile" ]; then
    echo "   ✅ Makefile"
else
    echo "   ❌ Makefile missing"
fi

if [ -f "README.md" ]; then
    echo "   ✅ README.md"
else
    echo "   ❌ README.md missing"
fi

echo ""
echo "=== Verification Complete ==="
echo ""
echo "🚀 To start the system:"
echo "   docker-compose up -d"
echo ""
echo "📖 For more information, see README.md"