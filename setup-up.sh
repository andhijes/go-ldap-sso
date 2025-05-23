#!/bin/bash

set -e  # Stop on error

# Step 1: Restart LDAP container
docker compose -f docker-compose.osixia.yml down -v
docker compose -f docker-compose.osixia.yml up --build -d

# Step 2: Wait until LDAP is ready
echo "⏳ Waiting for LDAP to become available..."
until docker exec $(docker ps -qf "name=openldap") ldapsearch -x -H ldap://localhost -b dc=example,dc=org -D "cn=admin,dc=example,dc=org" -w admin1234 > /dev/null 2>&1; do
  sleep 1
done
echo "✅ LDAP is ready."

# Step 3: Apply LDIF files
echo "🚀 Adding initial users and modifications..."
ldapadd -x -D "cn=admin,dc=example,dc=org" -w admin1234 -f ldif/001_init_user.ldif
ldapadd -x -D "cn=admin,dc=example,dc=org" -w admin1234 -f ldif/002_modify_john_email.ldif
ldapmodify -x -H ldap://localhost:389 -D "cn=admin,dc=example,dc=org" -w admin1234 -f ldif/003_replace_john_email.ldif

echo "✅ All done."
