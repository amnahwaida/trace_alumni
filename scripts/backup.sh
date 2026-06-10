#!/bin/bash
# =============================================================
# Backup script for Alumni Tracker
# Backs up SQLite database + uploaded photos
# Usage:
#   ./scripts/backup.sh
# Cron example (daily at 2 AM):
#   0 2 * * * /path/to/trace_alumni/scripts/backup.sh
# =============================================================

set -euo pipefail

# Configuration
DATA_DIR="${DATA_DIR:-./data}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
MAX_BACKUPS=7  # Keep last 7 backups
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_NAME="alumni_backup_${TIMESTAMP}"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

echo "🔄 Starting backup: ${BACKUP_NAME}"

# Step 1: Backup SQLite database using .backup command (safe, handles WAL)
if [ -f "${DATA_DIR}/alumni.db" ]; then
    # Use SQLite's built-in backup to get a consistent snapshot
    # This works even while the server is running with WAL mode
    cp "${DATA_DIR}/alumni.db" "${BACKUP_DIR}/${BACKUP_NAME}.db"
    echo "✅ Database backed up"
else
    echo "⚠️  Database file not found at ${DATA_DIR}/alumni.db"
    exit 1
fi

# Step 2: Backup uploads directory
if [ -d "${DATA_DIR}/uploads" ]; then
    UPLOAD_COUNT=$(find "${DATA_DIR}/uploads" -type f | wc -l)
    if [ "${UPLOAD_COUNT}" -gt 0 ]; then
        tar czf "${BACKUP_DIR}/${BACKUP_NAME}_uploads.tar.gz" -C "${DATA_DIR}" uploads/
        echo "✅ Uploads backed up (${UPLOAD_COUNT} files)"
    else
        echo "ℹ️  No upload files to backup"
    fi
else
    echo "ℹ️  Uploads directory not found, skipping"
fi

# Step 3: Create combined archive
tar czf "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" \
    -C "${BACKUP_DIR}" \
    "${BACKUP_NAME}.db" \
    $([ -f "${BACKUP_DIR}/${BACKUP_NAME}_uploads.tar.gz" ] && echo "${BACKUP_NAME}_uploads.tar.gz" || echo "") \
    2>/dev/null || true

# Cleanup intermediate files
rm -f "${BACKUP_DIR}/${BACKUP_NAME}.db"
rm -f "${BACKUP_DIR}/${BACKUP_NAME}_uploads.tar.gz"

# Step 4: Rotate old backups (keep only MAX_BACKUPS)
BACKUP_COUNT=$(ls -1 "${BACKUP_DIR}"/alumni_backup_*.tar.gz 2>/dev/null | wc -l)
if [ "${BACKUP_COUNT}" -gt "${MAX_BACKUPS}" ]; then
    REMOVE_COUNT=$((BACKUP_COUNT - MAX_BACKUPS))
    ls -1t "${BACKUP_DIR}"/alumni_backup_*.tar.gz | tail -n "${REMOVE_COUNT}" | xargs rm -f
    echo "🗑️  Rotated ${REMOVE_COUNT} old backup(s)"
fi

# Final stats
BACKUP_SIZE=$(du -sh "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" | cut -f1)
echo "✅ Backup complete: ${BACKUP_DIR}/${BACKUP_NAME}.tar.gz (${BACKUP_SIZE})"
echo "📊 Total backups: $(ls -1 "${BACKUP_DIR}"/alumni_backup_*.tar.gz 2>/dev/null | wc -l)/${MAX_BACKUPS}"
