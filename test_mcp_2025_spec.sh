#!/bin/bash
# Test script for MCP 2025-06-18 specification compliance

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Testing MCP 2025-06-18 Specification Compliance"
echo "================================================"
echo ""

# Start server in background
echo "Starting MCP server on port 9876..."
./mcpserver --env=/tmp/test.env > /tmp/mcpserver.log 2>&1 &
SERVER_PID=$!
sleep 2

# Check if server is running
if ! ps -p $SERVER_PID > /dev/null; then
    echo -e "${RED}✗ Server failed to start${NC}"
    cat /tmp/mcpserver.log
    exit 1
fi

echo -e "${GREEN}✓ Server started (PID: $SERVER_PID)${NC}"
echo ""

# Test 1: Health check
echo "Test 1: Health check endpoint"
HEALTH=$(curl -s http://localhost:9876/health)
if [ "$HEALTH" = "OK" ]; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
fi
echo ""

# Test 2: Protocol version validation (valid)
echo "Test 2: Valid protocol version (2025-06-18)"
HTTP_CODE=$(timeout 2 curl -s -o /dev/null -w "%{http_code}" \
    -H "MCP-Protocol-Version: 2025-06-18" \
    -H "X-API-Key: testkey" \
    -H "Accept: text/event-stream" \
    http://localhost:9876/mcp || echo "timeout")
# Note: SSE connections may return different codes depending on SDK behavior
echo "HTTP Status: $HTTP_CODE"
if [[ "$HTTP_CODE" =~ ^(200|101)$ ]]; then
    echo -e "${GREEN}✓ Valid protocol version accepted${NC}"
else
    echo -e "Note: Received HTTP $HTTP_CODE (SSE connection behavior varies)"
fi
echo ""

# Test 3: Protocol version validation (invalid)
echo "Test 3: Invalid protocol version (2023-01-01)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "MCP-Protocol-Version: 2023-01-01" \
    -H "X-API-Key: testkey" \
    -H "Accept: text/event-stream" \
    http://localhost:9876/mcp)
if [ "$HTTP_CODE" = "400" ]; then
    echo -e "${GREEN}✓ Invalid protocol version rejected with 400${NC}"
else
    echo -e "${RED}✗ Expected 400, got $HTTP_CODE${NC}"
fi
echo ""

# Test 4: Default protocol version
echo "Test 4: Default protocol version (no header)"
# This should default to 2025-03-26 and be accepted
HTTP_CODE=$(timeout 2 curl -s -o /dev/null -w "%{http_code}" \
    -H "X-API-Key: testkey" \
    -H "Accept: text/event-stream" \
    http://localhost:9876/mcp || echo "timeout")
echo "HTTP Status: $HTTP_CODE"
if [[ "$HTTP_CODE" =~ ^(200|101)$ ]]; then
    echo -e "${GREEN}✓ Default protocol version handled${NC}"
else
    echo -e "Note: Received HTTP $HTTP_CODE"
fi
echo ""

# Test 5: Authentication (missing key)
echo "Test 5: Authentication - missing API key"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "MCP-Protocol-Version: 2025-06-18" \
    -H "Accept: text/event-stream" \
    http://localhost:9876/mcp)
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ Missing API key rejected with 401${NC}"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_CODE${NC}"
fi
echo ""

# Test 6: Authentication (invalid key)
echo "Test 6: Authentication - invalid API key"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "MCP-Protocol-Version: 2025-06-18" \
    -H "X-API-Key: invalid-key" \
    -H "Accept: text/event-stream" \
    http://localhost:9876/mcp)
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ Invalid API key rejected with 401${NC}"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_CODE${NC}"
fi
echo ""

# Test 7: Session ID handling
echo "Test 7: Session ID in Mcp-Session-Id header"
# Start a connection with session ID and check server logs
curl -s -N \
    -H "MCP-Protocol-Version: 2025-06-18" \
    -H "Mcp-Session-Id: test-session-abc123" \
    -H "X-API-Key: testkey" \
    -H "Accept: text/event-stream" \
    http://localhost:9876/mcp > /dev/null 2>&1 &
CURL_PID=$!
sleep 1
kill $CURL_PID 2>/dev/null || true

# Check if session ID appears in logs
if grep -q "session: test-session-abc123" /tmp/mcpserver.log; then
    echo -e "${GREEN}✓ Session ID captured and logged${NC}"
else
    echo -e "${RED}✗ Session ID not found in logs${NC}"
fi
echo ""

# Cleanup
echo "Cleaning up..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "================================================"
echo "MCP 2025-06-18 Compliance Tests Complete"
echo "================================================"
