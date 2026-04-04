#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CERT_DIR="$PROJECT_ROOT/pb_frontend/nginx/cert"

mkdir -p "$CERT_DIR"

echo "=========================================="
echo "Generating SSL Certificates"
echo "=========================================="

if [ -f "$CERT_DIR/megapixelbattle.crt" ] && [ -f "$CERT_DIR/megapixelbattle.key" ]; then
    echo "Certificates already exist. Skipping generation."
    echo "To regenerate, delete existing certificates first."
    exit 0
fi

echo "Generating self-signed certificate..."
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$CERT_DIR/megapixelbattle.key" \
    -out "$CERT_DIR/megapixelbattle.crt" \
    -subj "/C=RU/ST=SPB/L=Saint-Petersburg/O=ITMO/CN=megapixelbattle.ru"

if [ $? -eq 0 ]; then
    echo "✓ Certificates generated successfully"
    echo "  Certificate: $CERT_DIR/megapixelbattle.crt"
    echo "  Private Key: $CERT_DIR/megapixelbattle.key"
    echo ""
    echo "Note: These are self-signed certificates for development only."
else
    echo "✗ Failed to generate certificates"
    exit 1
fi
