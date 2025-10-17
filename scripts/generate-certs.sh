#!/bin/bash

set -e

DOMAIN=""
OUTPUT_DIR="./certs"

show_help() {
    cat << EOF
Generate self-signed certificates for development

Usage: ./scripts/generate-certs.sh [OPTIONS]

Options:
  --domain DOMAIN       Domain name for certificate (required)
  --output-dir DIR      Output directory for certificates (default: ./certs)
  --help               Show this help

Examples:
  # Generate for localhost
  ./scripts/generate-certs.sh --domain localhost

  # Generate for custom domain and directory
  ./scripts/generate-certs.sh --domain photos.example.com --output-dir /tmp/certs

Note: Self-signed certificates are for DEVELOPMENT ONLY.
      For production, use Let's Encrypt or a trusted CA.

EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo "Error: Unknown option $1"
            show_help
            exit 1
            ;;
    esac
done

if [ -z "$DOMAIN" ]; then
    echo "Error: --domain is required"
    show_help
    exit 1
fi

echo "════════════════════════════════════════════════"
echo "  Generating Self-Signed Certificates"
echo "════════════════════════════════════════════════"
echo ""
echo "Domain: $DOMAIN"
echo "Output Directory: $OUTPUT_DIR"
echo ""
echo "WARNING: Self-signed certificates are for DEVELOPMENT only!"
echo ""

mkdir -p "$OUTPUT_DIR"

echo "Generating private key..."
openssl genrsa -out "$OUTPUT_DIR/privkey.pem" 4096

echo "Generating certificate..."
openssl req -new -x509 -key "$OUTPUT_DIR/privkey.pem" \
    -out "$OUTPUT_DIR/fullchain.pem" \
    -days 365 \
    -subj "/CN=$DOMAIN" \
    -addext "subjectAltName=DNS:$DOMAIN,DNS:localhost"

echo ""
echo "✓ Certificates generated successfully!"
echo ""
echo "Files created:"
echo "  $OUTPUT_DIR/privkey.pem"
echo "  $OUTPUT_DIR/fullchain.pem"
echo ""
echo "Certificate details:"
openssl x509 -in "$OUTPUT_DIR/fullchain.pem" -noout -dates
echo ""
echo "Next steps:"
echo "  1. Build Docker image with these certificates:"
echo "     ./scripts/build.sh --cert-path $OUTPUT_DIR --domain $DOMAIN"
echo ""
echo "  2. Configure your backend:"
echo "     cp examples/ad-smb.yaml config.yaml"
echo "     # Edit config.yaml"
echo ""
echo "  3. Deploy:"
echo "     ./scripts/deploy.sh --host local --image photo-sync-backend:latest"
echo ""

