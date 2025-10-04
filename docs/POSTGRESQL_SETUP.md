# PostgreSQL Setup Guide

## Overview

The simple-content-mcp server supports PostgreSQL as a production-ready repository backend, combined with filesystem or S3 storage for blob data.

## Architecture

- **Repository (Metadata)**: PostgreSQL stores content metadata, relationships, and indexes
- **Storage (Blobs)**: Filesystem or S3 stores the actual file data

## Prerequisites

- PostgreSQL 12+ installed and running
- Database created for simple-content

## Quick Start

### 1. Create PostgreSQL Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE simple_content;

# Create user (optional)
CREATE USER simple_content_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE simple_content TO simple_content_user;

# Exit psql
\q
```

### 2. Run Database Migrations

The simple-content library will automatically create required tables on first connection. The schema includes:

- `content` - Main content metadata table
- `content_tags` - Content tagging
- Indexes for efficient querying

### 3. Configure Environment

Create `.env.postgres`:

```bash
# Server
MCP_MODE=sse
MCP_HOST=localhost
MCP_PORT=3030

# PostgreSQL Database
DATABASE_URL=postgres://simple_content_user:your_password@localhost:5432/simple_content?sslmode=disable

# Filesystem Storage
STORAGE_BACKEND=fs
STORAGE_PATH=./data/storage

# Authentication
MCP_AUTH_ENABLED=true
MCP_API_KEY_1=your-key:550e8400-e29b-41d4-a716-446655440000::

# Features
MCP_ENABLE_RESOURCES=true
MCP_ENABLE_PROMPTS=true
```

### 4. Start Server

```bash
./mcpserver --env=.env.postgres
```

## Connection String Format

PostgreSQL connection strings follow the standard format:

```
postgres://[user[:password]@][host][:port]/database[?param1=value1&...]
```

### Examples

**Local development:**
```
postgres://user:password@localhost:5432/simple_content?sslmode=disable
```

**Production with SSL:**
```
postgres://user:password@db.example.com:5432/simple_content?sslmode=require
```

**Unix socket:**
```
postgres:///simple_content?host=/var/run/postgresql
```

**Connection pool settings:**
```
postgres://user:password@localhost:5432/simple_content?pool_max_conns=10&pool_min_conns=2
```

## Storage Backend Options

### Option 1: Filesystem Storage (Recommended for single-server deployments)

```bash
STORAGE_BACKEND=fs
STORAGE_PATH=./data/storage
STORAGE_URL_PREFIX=http://localhost:3030/files  # Optional
```

**Pros:**
- Simple setup
- No additional services required
- Good performance for local access

**Cons:**
- Not suitable for multi-server deployments
- Requires local disk space
- No built-in CDN support

### Option 2: S3 Storage (Recommended for production/cloud deployments)

```bash
STORAGE_BACKEND=s3
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
S3_BUCKET=your-content-bucket
```

**Pros:**
- Scalable across multiple servers
- Built-in redundancy
- CDN integration available
- No local disk usage

**Cons:**
- Requires AWS account
- Additional cost
- Network latency

## Database Schema

The PostgreSQL schema is managed by the simple-content library. Key tables:

```sql
-- Content metadata
CREATE TABLE content (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    tenant_id UUID,
    name VARCHAR(255) NOT NULL,
    file_name VARCHAR(255),
    content_type VARCHAR(100),
    size BIGINT,
    storage_key VARCHAR(500),
    status VARCHAR(50),
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX idx_content_owner ON content(owner_id);
CREATE INDEX idx_content_status ON content(status);
CREATE INDEX idx_content_tags ON content USING GIN(tags);
CREATE INDEX idx_content_created ON content(created_at);
```

## Performance Tuning

### PostgreSQL Configuration

For production deployments, tune these PostgreSQL settings:

```sql
-- postgresql.conf
shared_buffers = 256MB              # 25% of RAM
effective_cache_size = 1GB          # 50-75% of RAM
work_mem = 16MB                     # For sorting/joins
maintenance_work_mem = 128MB        # For VACUUM, CREATE INDEX
```

### Connection Pooling

The server uses pgxpool for connection pooling. Configure via connection string:

```
postgres://user:pass@host/db?pool_max_conns=25&pool_min_conns=5&pool_max_conn_lifetime=1h
```

Recommended settings:
- `pool_max_conns`: 2-3x number of CPU cores
- `pool_min_conns`: 2-5 connections
- `pool_max_conn_lifetime`: 1h (helps with connection refresh)

## Backup and Recovery

### Backup

```bash
# Full database backup
pg_dump -U simple_content_user simple_content > backup.sql

# Compressed backup
pg_dump -U simple_content_user simple_content | gzip > backup.sql.gz

# Backup with filesystem storage
tar -czf storage_backup.tar.gz ./data/storage
```

### Restore

```bash
# Restore database
psql -U simple_content_user simple_content < backup.sql

# Restore filesystem storage
tar -xzf storage_backup.tar.gz
```

## Monitoring

### Key Metrics

1. **Connection Pool Usage**
```sql
SELECT count(*) as connections, state
FROM pg_stat_activity
WHERE datname = 'simple_content'
GROUP BY state;
```

2. **Table Sizes**
```sql
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

3. **Index Usage**
```sql
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

## Troubleshooting

### Connection Failed

```
Error: failed to connect to PostgreSQL: connection refused
```

**Solutions:**
- Check PostgreSQL is running: `pg_isready`
- Verify connection string in `.env`
- Check firewall rules
- Verify `postgresql.conf` listen_addresses
- Check `pg_hba.conf` authentication rules

### Permission Denied

```
Error: permission denied for table content
```

**Solutions:**
```sql
GRANT ALL PRIVILEGES ON DATABASE simple_content TO simple_content_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO simple_content_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO simple_content_user;
```

### Too Many Connections

```
Error: too many connections for database
```

**Solutions:**
- Reduce `pool_max_conns` in connection string
- Increase PostgreSQL `max_connections` in postgresql.conf
- Use connection pooler like PgBouncer

## Production Deployment

### Recommended Setup

1. **High Availability**
   - PostgreSQL with replication (primary + replicas)
   - Load balancer for read queries
   - Automated failover (Patroni, Stolon)

2. **Monitoring**
   - pg_stat_statements for query performance
   - Connection pool metrics
   - Disk space monitoring

3. **Security**
   - SSL/TLS connections (sslmode=require)
   - Encrypted backups
   - Network isolation (VPC)
   - Regular security updates

4. **Scaling**
   - Read replicas for heavy read workloads
   - Connection pooling (PgBouncer/pgcat)
   - Partitioning for large tables (by date/tenant)

## See Also

- [Main README](../README.md)
- [Authentication Guide](AUTHENTICATION.md)
- [HTTP Streamable Transport](HTTP_STREAMABLE_TRANSPORT.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
