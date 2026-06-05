#!/bin/sh
set -e

echo "Running database migrations..."
for f in /app/migrations/*.sql; do
    echo "  Applying: $(basename $f)"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f "$f" 2>/dev/null || true
done

echo "Starting RMM API..."
exec /app/rmm-api
