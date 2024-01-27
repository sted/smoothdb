#!/bin/sh

export SMOOTHDB_DATABASE_URL="postgresql://${RDS_USERNAME}:${RDS_PASSWORD}@${RDS_HOSTNAME}:${RDS_PORT}"

exec /app/smoothdb