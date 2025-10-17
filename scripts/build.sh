#!/bin/bash

set -e

CERT_PATH=""
DOMAIN=""
IMAGE_TAG="photo-sync-backend:latest"
REGISTRY=""
CONFIG_FILE="config.yaml"

show_help() {
    cat << EOF
Build Photo Sync Backend Docker image

Usage: ./scripts/build.sh [OPTIONS]

Options:
  --cert-path PATH      Path to certificate directory containing fullchain.pem and privkey.pem (required)
  --domain DOMAIN       Domain name for certificate validation (required)
  --image-tag TAG       Docker image tag (default: photo-sync-backend:latest)
  --registry REGISTRY   Push to registry after build (optional)
  --config FILE         Config file to check/update JWT keys (default: config.yaml)
  --help               Show this help

Examples:
  # Build with Let's Encrypt certificates
  ./scripts/build.sh \\
    --cert-path /etc/letsencrypt/live/yourdomain.com \\
    --domain yourdomain.com \\
    --image-tag photo-sync-backend:v1.0

  # Build and push to registry
  ./scripts/build.sh \\
    --cert-path ./certs \\
    --domain yourdomain.com \\
    --image-tag myregistry.com/photo-sync:v1.0 \\
    --registry myregistry.com

  # Build with self-signed certificates
  ./scripts/generate-certs.sh --domain localhost
  ./scripts/build.sh \\
    --cert-path ./certs \\
    --domain localhost \\
    --image-tag photo-sync-backend:dev

EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --cert-path)
            CERT_PATH="$2"
            shift 2
            ;;
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --image-tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --config)
            CONFIG_FILE="$2"
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

if [ -z "$CERT_PATH" ]; then
    echo "Error: --cert-path is required"
    show_help
    exit 1
fi

if [ -z "$DOMAIN" ]; then
    echo "Error: --domain is required"
    show_help
    exit 1
fi

echo "════════════════════════════════════════════════"
echo "  Building Photo Sync Backend Docker Image"
echo "════════════════════════════════════════════════"
echo ""
echo "Configuration:"
echo "  Certificate Path: $CERT_PATH"
echo "  Domain: $DOMAIN"
echo "  Image Tag: $IMAGE_TAG"
if [ -n "$REGISTRY" ]; then
    echo "  Registry: $REGISTRY"
fi
echo ""

if [ ! -d "$CERT_PATH" ]; then
    echo "Error: Certificate directory does not exist: $CERT_PATH"
    exit 1
fi

if [ ! -f "$CERT_PATH/fullchain.pem" ]; then
    echo "Error: fullchain.pem not found in $CERT_PATH"
    exit 1
fi

if [ ! -f "$CERT_PATH/privkey.pem" ]; then
    echo "Error: privkey.pem not found in $CERT_PATH"
    exit 1
fi

echo "✓ Certificates found"
echo ""

echo "Validating certificate for domain: $DOMAIN"
if openssl x509 -in "$CERT_PATH/fullchain.pem" -noout -text | grep -q "CN.*$DOMAIN"; then
    echo "✓ Certificate matches domain"
else
    echo "Warning: Certificate may not match domain $DOMAIN"
    echo "Continuing anyway..."
fi

CERT_DATES=$(openssl x509 -in "$CERT_PATH/fullchain.pem" -noout -dates)
echo "$CERT_DATES"
echo ""

mkdir -p certs
echo "Copying certificates to build context..."
cp "$CERT_PATH/fullchain.pem" certs/
cp "$CERT_PATH/privkey.pem" certs/
echo "✓ Certificates copied"
echo ""

generate_jwt_keys() {
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "Config file $CONFIG_FILE not found, skipping JWT key generation"
        return
    fi

    echo "Checking JWT keys in $CONFIG_FILE..."

    # Check if JWT keys need generation
    SECRET_KEY=$(grep "secret_key:" "$CONFIG_FILE" | sed 's/.*secret_key: *//' | tr -d '"' || echo "")
    ENCRYPTION_KEY=$(grep "encryption_key:" "$CONFIG_FILE" | sed 's/.*encryption_key: *//' | tr -d '"' || echo "")

    NEEDS_GENERATION=false

    if [ -z "$SECRET_KEY" ] || [[ "$SECRET_KEY" == *"CHANGE"* ]] || [[ "$SECRET_KEY" == *"PLACEHOLDER"* ]] || [ ${#SECRET_KEY} -lt 32 ]; then
        NEEDS_GENERATION=true
        echo "  → Secret key needs generation"
    fi

    if [ -z "$ENCRYPTION_KEY" ] || [[ "$ENCRYPTION_KEY" == *"CHANGE"* ]] || [[ "$ENCRYPTION_KEY" == *"PLACEHOLDER"* ]] || [ ${#ENCRYPTION_KEY} -ne 32 ]; then
        NEEDS_GENERATION=true
        echo "  → Encryption key needs generation"
    fi

    if [ "$NEEDS_GENERATION" = true ]; then
        echo "Generating new JWT keys..."

        NEW_SECRET_KEY=$(openssl rand -base64 64)
        NEW_ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)

        # Backup original file
        cp "$CONFIG_FILE" "${CONFIG_FILE}.backup"

        # Write keys to temporary files to avoid shell escaping issues
        echo "$NEW_SECRET_KEY" > /tmp/jwt_secret_key.tmp
        echo "$NEW_ENCRYPTION_KEY" > /tmp/jwt_encryption_key.tmp

        # Update the keys in config file using Python for safe replacement
        python3 -c "
import re
import sys

with open('$CONFIG_FILE', 'r') as f:
    content = f.read()

with open('/tmp/jwt_secret_key.tmp', 'r') as f:
    secret_key = f.read().strip()

with open('/tmp/jwt_encryption_key.tmp', 'r') as f:
    encryption_key = f.read().strip()

# Replace secret key
content = re.sub(r'(\s+secret_key:\s*)\"[^\"]*\"', f'\g<1>\"{secret_key}\"', content)

# Replace encryption key
content = re.sub(r'(\s+encryption_key:\s*)\"[^\"]*\"', f'\g<1>\"{encryption_key}\"', content)

with open('${CONFIG_FILE}.tmp', 'w') as f:
    f.write(content)
" && mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"

        # Clean up temporary files
        rm -f /tmp/jwt_secret_key.tmp /tmp/jwt_encryption_key.tmp

        echo "✓ JWT keys generated and updated in $CONFIG_FILE"
        echo "  → Backup saved as ${CONFIG_FILE}.backup"
        echo ""
        echo "⚠️  IMPORTANT: Save these keys securely!"
        echo "   Secret Key: $NEW_SECRET_KEY"
        echo "   Encryption Key: $NEW_ENCRYPTION_KEY"
        echo ""
        echo "   These keys are used to sign and encrypt JWT tokens."
        echo "   If lost, all users will need to log in again."
        echo ""
    else
        echo "✓ JWT keys are already configured"
    fi
}

generate_jwt_keys

echo "Building Docker image: $IMAGE_TAG"
docker build -t "$IMAGE_TAG" .

echo ""
echo "✓ Docker image built successfully"
echo ""

if [ -n "$REGISTRY" ]; then
    echo "Pushing to registry: $REGISTRY"
    docker push "$IMAGE_TAG"
    echo "✓ Image pushed to registry"
    echo ""
fi

echo "════════════════════════════════════════════════"
echo "  Build Complete!"
echo "════════════════════════════════════════════════"
echo ""
echo "Image: $IMAGE_TAG"
echo ""
echo "Next steps:"
echo "  1. Configure your backend:"
echo "     cp examples/ad-smb.yaml config.yaml"
echo "     # Edit config.yaml with your settings"
echo ""
echo "  2. Deploy the container:"
echo "     ./scripts/deploy.sh --host local --image $IMAGE_TAG --config config.yaml"
echo ""

if [ -n "$REGISTRY" ]; then
    echo "  Or deploy to remote host:"
    echo "     ./scripts/deploy.sh --host 192.168.1.100 --ssh-key ~/.ssh/id_rsa --image $IMAGE_TAG"
    echo ""
fi

echo "Note: Certificates expire in 90 days. Remember to rebuild with updated certificates."
echo ""

