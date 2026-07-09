#!/bin/bash
set -e

echo "🧪 Testing Prometheus Remote Write"
echo "==================================="
echo ""

# Check if docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

echo "📦 Starting VictoriaMetrics..."
docker run -d \
  --name victoriametrics-test \
  --rm \
  -p 8428:8428 \
  victoriametrics/victoria-metrics:latest

# Wait for VictoriaMetrics to be ready
echo "⏳ Waiting for VictoriaMetrics to start..."
sleep 3

# Check if VictoriaMetrics is healthy
if curl -sf http://localhost:8428/health > /dev/null; then
    echo "✅ VictoriaMetrics is running"
else
    echo "❌ VictoriaMetrics failed to start"
    docker logs victoriametrics-test
    docker stop victoriametrics-test 2>/dev/null || true
    exit 1
fi

echo ""
echo "🚀 Ready to test!"
echo ""
echo "Run solar-controller with:"
echo "  make build-backend"
echo "  ./bin/solar-controller -config examples/remotewrite/config.yaml"
echo ""
echo "View metrics at: http://localhost:8428/vmui"
echo ""
echo "Stop VictoriaMetrics: docker stop victoriametrics-test"
echo ""
echo "Following VictoriaMetrics logs (Ctrl+C to exit):"
echo ""

# Follow logs
docker logs -f victoriametrics-test
