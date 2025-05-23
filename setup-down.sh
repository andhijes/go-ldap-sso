#!/bin/bash

set -e  # Stop if any command fails

echo "🛑 Stopping and removing LDAP container and volumes..."
docker compose -f docker-compose.osixia.yml down -v

echo "✅ Container and volumes removed."
