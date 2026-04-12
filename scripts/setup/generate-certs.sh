#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CERT_DIR="$PROJECT_ROOT/certs"

mkdir -p "$CERT_DIR"

echo "=========================================="
echo "Generating self-signed TLS certificates"
echo "=========================================="

if [[ -f "$CERT_DIR/pixelbattle.crt" && -f "$CERT_DIR/pixelbattle.key" ]]; then
	echo "Certificates already exist. Skipping generation."
	echo "To regenerate, delete them first."
	exit 0
fi

# Optional: public IP or hostname for SAN (avoids browser warnings vs CN-only localhost)
# Usage: ./generate-certs.sh
#        PUBLIC_IP=203.0.113.10 ./generate-certs.sh
#        ./generate-certs.sh 203.0.113.10
SAN="DNS:localhost"
if [[ "${1:-}" =~ ^[0-9.]+$ ]]; then
	SAN="IP:${1},DNS:localhost"
elif [[ -n "${PUBLIC_IP:-}" ]]; then
	SAN="IP:${PUBLIC_IP},DNS:localhost"
fi

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
	-keyout "$CERT_DIR/pixelbattle.key" \
	-out "$CERT_DIR/pixelbattle.crt" \
	-subj "/CN=localhost" \
	-addext "subjectAltName=${SAN}"

echo "Certificates written to:"
echo "  $CERT_DIR/pixelbattle.crt"
echo "  $CERT_DIR/pixelbattle.key"
echo "  SAN: ${SAN}"
echo ""
echo "Note: self-signed; browsers will show a warning unless you trust the CA."
