#!/bin/bash

# ============================================================================
# Database Service Backup Script
# ============================================================================
# Supports: MongoDB, Redis, Vault data backup
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
MONGODB_HOST="${MONGODB_HOST:-localhost}"
MONGODB_PORT="${MONGODB_PORT:-27017}"
MONGODB_USER="${MONGODB_USER:-admin}"
MONGODB_PASS="${MONGODB_PASS:-password}"
MONGODB_DB="${MONGODB_DB:-testdb}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASS="${REDIS_PASS:-redispassword}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Functions
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

create_backup_dir() {
    local dir="$1"
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        print_info "Created backup directory: $dir"
    fi
}

# MongoDB Backup
backup_mongodb() {
    print_info "Starting MongoDB backup..."

    local backup_path="$BACKUP_DIR/mongodb_${TIMESTAMP}"
    create_backup_dir "$backup_path"

    # Check if mongodump is available
    if ! command -v mongodump &> /dev/null; then
        print_warning "mongodump not found, trying docker..."

        docker exec database-service-mongodb mongodump \
            --uri="mongodb://$MONGODB_USER:$MONGODB_PASS@localhost:27017/$MONGODB_DB?authSource=admin" \
            --out="/tmp/backup_${TIMESTAMP}" \
            --gzip

        docker cp "database-service-mongodb:/tmp/backup_${TIMESTAMP}" "$backup_path"
        docker exec database-service-mongodb rm -rf "/tmp/backup_${TIMESTAMP}"
    else
        mongodump \
            --host="$MONGODB_HOST" \
            --port="$MONGODB_PORT" \
            --username="$MONGODB_USER" \
            --password="$MONGODB_PASS" \
            --db="$MONGODB_DB" \
            --out="$backup_path" \
            --gzip
    fi

    # Create tarball
    tar -czf "${backup_path}.tar.gz" -C "$BACKUP_DIR" "mongodb_${TIMESTAMP}"
    rm -rf "$backup_path"

    local size=$(du -h "${backup_path}.tar.gz" | cut -f1)
    print_info "✅ MongoDB backup completed: ${backup_path}.tar.gz ($size)"
}

# Redis Backup
backup_redis() {
    print_info "Starting Redis backup..."

    local backup_path="$BACKUP_DIR/redis_${TIMESTAMP}"
    create_backup_dir "$backup_path"

    # Trigger Redis BGSAVE
    if command -v redis-cli &> /dev/null; then
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASS" BGSAVE
        sleep 2

        # Copy dump.rdb
        docker cp database-service-redis:/data/dump.rdb "${backup_path}/dump.rdb"
    else
        docker exec database-service-redis redis-cli -a "$REDIS_PASS" BGSAVE
        sleep 2
        docker cp database-service-redis:/data/dump.rdb "${backup_path}/dump.rdb"
    fi

    # Create tarball
    tar -czf "${backup_path}.tar.gz" -C "$BACKUP_DIR" "redis_${TIMESTAMP}"
    rm -rf "$backup_path"

    local size=$(du -h "${backup_path}.tar.gz" | cut -f1)
    print_info "✅ Redis backup completed: ${backup_path}.tar.gz ($size)"
}

# Application Data Backup (configs, logs)
backup_app_data() {
    print_info "Starting application data backup..."

    local backup_path="$BACKUP_DIR/appdata_${TIMESTAMP}"
    create_backup_dir "$backup_path"

    # Backup configs
    if [ -d "./configs" ]; then
        cp -r ./configs "$backup_path/"
    fi

    # Backup logs (if exist)
    if [ -d "./logs" ]; then
        cp -r ./logs "$backup_path/"
    fi

    # Create tarball
    tar -czf "${backup_path}.tar.gz" -C "$BACKUP_DIR" "appdata_${TIMESTAMP}"
    rm -rf "$backup_path"

    local size=$(du -h "${backup_path}.tar.gz" | cut -f1)
    print_info "✅ Application data backup completed: ${backup_path}.tar.gz ($size)"
}

# Clean old backups
cleanup_old_backups() {
    print_info "Cleaning up backups older than $RETENTION_DAYS days..."

    find "$BACKUP_DIR" -name "*.tar.gz" -type f -mtime +$RETENTION_DAYS -delete

    local count=$(find "$BACKUP_DIR" -name "*.tar.gz" -type f | wc -l)
    print_info "✅ Cleanup completed. $count backup(s) remaining"
}

# Create backup manifest
create_manifest() {
    local manifest_file="$BACKUP_DIR/backup_${TIMESTAMP}_manifest.txt"

    cat > "$manifest_file" << EOF
Backup Manifest
===============
Date: $(date)
Host: $(hostname)
Version: 1.0

MongoDB:
  Host: $MONGODB_HOST:$MONGODB_PORT
  Database: $MONGODB_DB
  File: mongodb_${TIMESTAMP}.tar.gz

Redis:
  Host: $REDIS_HOST:$REDIS_PORT
  File: redis_${TIMESTAMP}.tar.gz

Application Data:
  File: appdata_${TIMESTAMP}.tar.gz

Retention: $RETENTION_DAYS days
EOF

    print_info "✅ Backup manifest created: $manifest_file"
}

# Main backup process
main() {
    print_info "========================================="
    print_info "Database Service Backup"
    print_info "========================================="
    print_info "Timestamp: $TIMESTAMP"
    print_info "Backup directory: $BACKUP_DIR"
    print_info ""

    create_backup_dir "$BACKUP_DIR"

    # Perform backups
    backup_mongodb
    backup_redis
    backup_app_data

    # Create manifest
    create_manifest

    # Cleanup
    cleanup_old_backups

    print_info ""
    print_info "========================================="
    print_info "✅ Backup completed successfully!"
    print_info "========================================="
    print_info "Backup files are located in: $BACKUP_DIR"
    print_info ""
    print_info "To restore:"
    print_info "  ./scripts/restore.sh $TIMESTAMP"
}

# Run main
main
