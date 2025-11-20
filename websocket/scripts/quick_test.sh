#!/bin/bash

set -e

echo "🧪 WebSocket Service Quick Test"
echo "================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Redis is running
echo "1️⃣  Checking Redis..."
if redis-cli -a 21042004 ping > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Redis is running${NC}"
else
    echo -e "${RED}❌ Redis is not running!${NC}"
    echo -e "${YELLOW}Please start Redis first:${NC}"
    echo "   redis-server --requirepass 21042004"
    echo "   OR"
    echo "   docker run -d -p 6379:6379 redis:7-alpine redis-server --requirepass 21042004"
    exit 1
fi

echo ""
echo "2️⃣  Generating JWT token..."
TOKEN=$(go run generate_token.go | grep "eyJ" | head -1)
echo -e "${GREEN}✅ Token generated${NC}"
echo "Token: $TOKEN"

echo ""
echo "3️⃣  Building service..."
go build -o websocket-server ./cmd/server
echo -e "${GREEN}✅ Build successful${NC}"

echo ""
echo "4️⃣  Starting WebSocket service..."
echo -e "${YELLOW}(The service will start in the background)${NC}"
./websocket-server > server.log 2>&1 &
SERVER_PID=$!
echo -e "${GREEN}✅ Server started (PID: $SERVER_PID)${NC}"

# Wait for server to start
sleep 2

echo ""
echo "5️⃣  Testing health endpoint..."
HEALTH=$(curl -s http://localhost:8081/health | jq -r '.status')
if [ "$HEALTH" = "healthy" ]; then
    echo -e "${GREEN}✅ Health check passed${NC}"
else
    echo -e "${RED}❌ Health check failed${NC}"
    kill $SERVER_PID
    exit 1
fi

echo ""
echo "6️⃣  Testing metrics endpoint..."
METRICS=$(curl -s http://localhost:8081/metrics | jq -r '.service')
if [ "$METRICS" = "websocket-service" ]; then
    echo -e "${GREEN}✅ Metrics endpoint working${NC}"
else
    echo -e "${RED}❌ Metrics endpoint failed${NC}"
    kill $SERVER_PID
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}🎉 All tests passed!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📡 WebSocket Service Info:"
echo "   • Server PID: $SERVER_PID"
echo "   • Health: http://localhost:8081/health"
echo "   • Metrics: http://localhost:8081/metrics"
echo "   • WebSocket: ws://localhost:8081/ws?token=TOKEN"
echo ""
echo "🧪 To test WebSocket connection:"
echo "   Terminal 1 (Already running): Watch server logs"
echo "      tail -f server.log"
echo ""
echo "   Terminal 2: Connect WebSocket client"
echo "      go run tests/client_example.go $TOKEN"
echo ""
echo "   Terminal 3: Send test message"
echo "      redis-cli -a 21042004"
echo "      PUBLISH user_noti:user123 '{\"type\":\"notification\",\"payload\":{\"title\":\"Hello\",\"body\":\"Test\"}}'"
echo ""
echo "🛑 To stop the server:"
echo "   kill $SERVER_PID"
echo ""
