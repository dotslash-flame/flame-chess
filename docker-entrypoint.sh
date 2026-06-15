#!/bin/sh
set -e

echo "running migrations..."
goose -dir /migrations postgres "$DATABASE_URL" up

echo "starting server..."
exec /server
