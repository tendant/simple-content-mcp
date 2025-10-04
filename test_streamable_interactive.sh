#!/bin/bash
# Interactive HTTP Streamable Transport Test

set -e

PORT=${MCP_PORT:-3030}
BASE_URL="http://localhost:$PORT"
OWNER_ID="550e8400-e29b-41d4-a716-446655440000"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  HTTP Streamable Transport Interactive Test   ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""

# Check if server is running
echo -n "Checking if server is running... "
if ! curl -s --max-time 2 $BASE_URL/health > /dev/null 2>&1; then
    echo -e "${YELLOW}✗${NC}"
    echo ""
    echo "Server not running on port $PORT."
    echo "Start server with: make run-stream"
    echo "Or: ./mcpserver --mode=sse --port=$PORT"
    exit 1
fi
echo -e "${GREEN}✓${NC}"

# Health checks
echo -n "Health check... "
HEALTH=$(curl -s $BASE_URL/health)
[ "$HEALTH" = "OK" ] && echo -e "${GREEN}✓${NC}" || echo -e "${YELLOW}✗${NC}"

echo -n "Ready check... "
READY=$(curl -s $BASE_URL/ready)
[ "$READY" = "READY" ] && echo -e "${GREEN}✓${NC}" || echo -e "${YELLOW}✗${NC}"

echo ""
echo -e "${BLUE}Step 1: Opening SSE Connection${NC}"
echo "This will open an SSE stream in the background..."

# Create a temporary file for session endpoint
SESSION_FILE=$(mktemp)

# Open SSE connection in background and capture session endpoint
(
    curl -N -s \
        -H "Accept: text/event-stream" \
        -H "MCP-Protocol-Version: 2025-06-18" \
        $BASE_URL/mcp 2>/dev/null | \
    while IFS= read -r line; do
        if [[ $line == data:* ]]; then
            SESSION_ENDPOINT="${line#data: }"
            echo "$SESSION_ENDPOINT" > "$SESSION_FILE"
            echo -e "${GREEN}Session endpoint received: $SESSION_ENDPOINT${NC}"
        fi
        echo "$line"
    done
) &
SSE_PID=$!

# Wait for session endpoint
echo "Waiting for session endpoint..."
sleep 2

if [ ! -s "$SESSION_FILE" ]; then
    echo -e "${YELLOW}Failed to get session endpoint${NC}"
    kill $SSE_PID 2>/dev/null
    rm -f "$SESSION_FILE"
    exit 1
fi

SESSION_ENDPOINT=$(cat "$SESSION_FILE")
echo -e "${GREEN}Got session: $SESSION_ENDPOINT${NC}"
echo ""

# Function to send JSON-RPC request
send_request() {
    local method=$1
    local params=$2
    local id=$3

    echo -e "${BLUE}Sending: $method${NC}"

    curl -s -X POST "$BASE_URL$SESSION_ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -H "MCP-Protocol-Version: 2025-06-18" \
        -d "{
            \"jsonrpc\": \"2.0\",
            \"id\": $id,
            \"method\": \"$method\",
            \"params\": $params
        }" | jq '.' 2>/dev/null || echo "(Response will appear in SSE stream above)"

    sleep 1
}

echo -e "${BLUE}Step 2: Initialize MCP Session${NC}"
echo ""

# IMPORTANT: MCP requires initialization before any other requests
echo -e "${YELLOW}Initializing MCP session...${NC}"
send_request "initialize" "{
    \"protocolVersion\": \"2025-06-18\",
    \"capabilities\": {
        \"tools\": {},
        \"resources\": {},
        \"prompts\": {}
    },
    \"clientInfo\": {
        \"name\": \"test-client\",
        \"version\": \"1.0.0\"
    }
}" 0
echo ""

# After initialization, send initialized notification
echo -e "${YELLOW}Sending initialized notification...${NC}"
curl -s -X POST "$BASE_URL$SESSION_ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "MCP-Protocol-Version: 2025-06-18" \
    -d '{
        "jsonrpc": "2.0",
        "method": "notifications/initialized"
    }' > /dev/null
sleep 1
echo ""

echo -e "${BLUE}Step 3: Testing Tools${NC}"
echo ""

# Test 1: List tools
echo -e "${YELLOW}Test 1: List available tools${NC}"
send_request "tools/list" "{}" 1
echo ""

# Test 2: Upload content
echo -e "${YELLOW}Test 2: Upload content${NC}"
TEST_DATA=$(echo "Hello from HTTP Streamable test!" | base64)
send_request "tools/call" "{
    \"name\": \"upload_content\",
    \"arguments\": {
        \"owner_id\": \"$OWNER_ID\",
        \"name\": \"test-file.txt\",
        \"data\": \"$TEST_DATA\",
        \"file_name\": \"test-file.txt\",
        \"tags\": [\"test\", \"http-streamable\"]
    }
}" 2
echo ""

# Test 3: List content
echo -e "${YELLOW}Test 3: List uploaded content${NC}"
send_request "tools/call" "{
    \"name\": \"list_content\",
    \"arguments\": {
        \"owner_id\": \"$OWNER_ID\",
        \"limit\": 5
    }
}" 3
echo ""

# Test 4: Get resources
echo -e "${YELLOW}Test 4: Read stats resource${NC}"
send_request "resources/read" "{
    \"uri\": \"stats://system\"
}" 4
echo ""

# Test 5: List prompts
echo -e "${YELLOW}Test 5: List available prompts${NC}"
send_request "prompts/list" "{}" 5
echo ""

echo -e "${GREEN}Test complete!${NC}"
echo ""
echo "The SSE connection is still open in the background (PID: $SSE_PID)"
echo "You can send additional requests manually:"
echo ""
echo -e "${BLUE}Example:${NC}"
echo "curl -X POST '$BASE_URL$SESSION_ENDPOINT' \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'MCP-Protocol-Version: 2025-06-18' \\"
echo "  -d '{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"tools/list\"}'"
echo ""
echo "Press Enter to close SSE connection and exit..."
read

# Cleanup
kill $SSE_PID 2>/dev/null || true
rm -f "$SESSION_FILE"

echo "Done!"
