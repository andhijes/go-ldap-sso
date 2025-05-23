#!/bin/bash
set -e

# Copy LDIF files if ada
if compgen -G "/container/ldif_input/*.ldif" > /dev/null; then
  echo "Copying LDIF files..."
  cp /container/ldif_input/*.ldif /container/service/slapd/assets/config/bootstrap/ldif/
else
  echo "No LDIF files found to copy."
fi

# Jalankan startup OpenLDAP bawaan image
/container/run/startup.sh
