#!/bin/sh

echo "🕒 Waiting for PostgreSQL to be ready on openldap_postgres:5432..."

until nc -z openldap_postgres 5432; do
  echo "⏳ Waiting for PostgreSQL..."
  sleep 2
done

echo "✅ PostgreSQL is ready."

# Execute the original entrypoint
exec /container/service/slapd/startup-original.sh