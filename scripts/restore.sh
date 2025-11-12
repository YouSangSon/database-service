#!/bin/bash

# ============================================================================
# Database Service Restore Script
# ============================================================================
# Supports: MongoDB, Redis, Vault data restore
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
MONGODB_HOST="${MONGODB_HOST:-localhost}"
MONGODB_PORT="${MONGODB_PORT:-27017}"
MONGODB_USER="${MONGODB_USER:-admin}"
MONGODB_PASS="${MONGODB_PASS:-password}"
MONGODB_DB="${MONGODB_DB:-testdb}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASS="${REDIS_PASS:-redispassword}"

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

# List available backups
list_backups() {
    print_info "Available backups:"
    echo ""

    if [ ! -d "$BACKUP_DIR" ]; then
        print_error "Backup directory not found: $BACKUP_DIR"
        exit 1
    fi

    local manifests=$(find "$BACKUP_DIR" -name "*_manifest.txt" | sort -r)

    if [ -z "$manifests" ]; then
        print_warning "No backups found in $BACKUP_DIR"
        exit 1
    fi

    for manifest in $manifests; do
        local timestamp=$(basename "$manifest" | sed 's/backup_\(.*\)_manifest.txt/\1/')
        local date=$(grep "Date:" "$manifest" | cut -d: -f2-)
        echo "  📦 $timestamp -$date"
    done

    echo ""
}

# Restore MongoDB
restore_mongodb() {
    local timestamp="$1"
    local backup_file="$BACKUP_DIR/mongodb_${timestamp}.tar.gz"

    if [ ! -f "$backup_file" ]; then
        print_error "MongoDB backup not found: $backup_file"
        return 1
    fi

    print_info "Restoring MongoDB from $backup_file..."

    # Extract backup
    local temp_dir=$(mktemp -d)
    tar -xzf "$backup_file" -C "$temp_dir"

    # Restore using mongorestore
    if command -v mongorestore &> /dev/null; then
        mongorestore \
            --host="$MONGODB_HOST" \
            --port="$MONGODB_PORT" \
            --username="$MONGODB_USER" \
            --password="$MONGODB_PASS" \
            --db="$MONGODB_DB" \
            --gzip \
            --drop \
            "$temp_dir/mongodb_${timestamp}/$MONGODB_DB"
    else
        # Copy to container and restore
        docker cp "$temp_dir/mongodb_${timestamp}" database-service-mongodb:/tmp/restore
        docker exec database-service-mongodb mongorestore \
            --uri="mongodb://$MONGODB_USER:$MONGODB_PASS@localhost:27017/$MONGODB_DB?authSource=admin" \
            --gzip \
            --drop \
            "/tmp/restore/$MONGODB_DB"
        docker exec database-service-mongodb rm -rf /tmp/restore
    fi

    rm -rf "$temp_dir"
    print_info "✅ MongoDB restore completed"
}

# Restore Redis
restore_redis() {
    local timestamp="$1"
    local backup_file="$BACKUP_DIR/redis_${timestamp}.tar.gz"

    if [ ! -f "$backup_file" ]; then
        print_error "Redis backup not found: $backup_file"
        return 1
    fi

    print_info "Restoring Redis from $backup_file..."

    # Stop Redis (to replace dump.rdb)
    print_warning "Stopping Redis..."
    docker stop database-service-redis

    # Extract and copy dump.rdb
    local temp_dir=$(mktemp -d)
    tar -xzf "$backup_file" -C "$temp_dir"

    # Copy dump.rdb to Redis data directory
    docker cp "${temp_dir}/redis_${timestamp}/dump.rdb" database-service-redis:/data/dump.rdb

    # Start Redis
    print_info "Starting Redis..."
    docker start database-service-redis

    # Wait for Redis to be ready
    sleep 3

    rm -rf "$temp_dir"
    print_info "✅ Redis restore completed"
}

# Restore application data
restore_app_data() {
    local timestamp="$1"
    local backup_file="$BACKUP_DIR/appdata_${timestamp}.tar.gz"

    if [ ! -f "$backup_file" ]; then
        print_warning "Application data backup not found: $backup_file"
        return 0
    fi

    print_info "Restoring application data from $backup_file..."

    # Create backup of current configs
    if [ -d "./configs" ]; then
        cp -r ./configs "./configs.backup.$(date +%Y%m%d_%H%M%S)"
    fi

    # Extract backup
    local temp_dir=$(mktemp -d)
    tar -xzf "$backup_file" -C "$temp_dir"

    # Restore configs
    if [ -d "$temp_dir/appdata_${timestamp}/configs" ]; then
        cp -r "$temp_dir/appdata_${timestamp}/configs" ./
    fi

    rm -rf "$temp_dir"
    print_info "✅ Application data restore completed"
}

# Confirm restore
confirm_restore() {
    local timestamp="$1"

    print_warning "⚠️  This will REPLACE all current data with backup from: $timestamp"
    print_warning "⚠️  Current data will be LOST unless you have another backup"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        print_info "Restore cancelled"
        exit 0
    fi
}

# Main restore process
main() {
    local timestamp="$1"

    print_info "========================================="
    print_info "Database Service Restore"
    print_info "========================================="
    print_info ""

    # List backups if no timestamp provided
    if [ -z "$timestamp" ]; then
        list_backups
        print_info "Usage: $0 <timestamp>"
        print_info "Example: $0 20250112_153045"
        exit 1
    fi

    # Check if backup exists
    local manifest_file="$BACKUP_DIR/backup_${timestamp}_manifest.txt"
    if [ ! -f "$manifest_file" ]; then
        print_error "Backup not found: $timestamp"
        list_backups
        exit 1
    fi

    # Show backup info
    print_info "Backup information:"
    cat "$manifest_file"
    echo ""

    # Confirm
    confirm_restore "$timestamp"

    print_info ""
    print_info "Starting restore process..."
    print_info ""

    # Perform restore
    restore_mongodb "$timestamp"
    restore_redis "$timestamp"
    restore_app_data "$timestamp"

    print_info ""
    print_info "========================================="
    print_info "✅ Restore completed successfully!"
    print_info "========================================="
    print_info "Please verify that services are working correctly:"
    print_info "  docker-compose ps"
    print_info "  curl http://localhost:8080/health"
}

# Run main
main "$@"
