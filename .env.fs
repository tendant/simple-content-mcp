# Filesystem Storage Configuration (with in-memory database)

# Server mode
MCP_MODE=sse
MCP_HOST=localhost
MCP_PORT=3030

# No DATABASE_URL = use in-memory repository

# Storage - Filesystem
STORAGE_BACKEND=fs
STORAGE_PATH=./data/storage

# Features
MCP_ENABLE_RESOURCES=true
MCP_ENABLE_PROMPTS=true
