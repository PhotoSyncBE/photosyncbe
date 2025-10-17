#!/bin/bash

set -e

HOST="local"
PORT="8443"
IMAGE=""
CONFIG="config.yaml"
SSH_KEY=""

show_help() {
    cat << EOF
Deploy Photo Sync Backend container

Usage: ./scripts/deploy.sh [OPTIONS]

Options:
  --host HOST          Docker host address or "local" (default: local)
  --port PORT          Service port (default: 8443)
  --image IMAGE        Docker image to deploy (required)
  --config CONFIG      Path to config.yaml (default: config.yaml)
  --ssh-key KEY        SSH key for remote host deployment (required for remote)
  --help              Show this help

Examples:
  # Deploy locally
  ./scripts/deploy.sh \\
    --host local \\
    --image photo-sync-backend:v1.0 \\
    --config config.yaml

  # Deploy to remote host
  ./scripts/deploy.sh \\
    --host 192.168.1.100 \\
    --ssh-key ~/.ssh/id_rsa \\
    --image myregistry.com/photo-sync:v1.0 \\
    --config config.yaml

  # Deploy with custom port
  ./scripts/deploy.sh \\
    --host local \\
    --port 9443 \\
    --image photo-sync-backend:latest

EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            HOST="$2"
            shift 2
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --image)
            IMAGE="$2"
            shift 2
            ;;
        --config)
            CONFIG="$2"
            shift 2
            ;;
        --ssh-key)
            SSH_KEY="$2"
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

if [ -z "$IMAGE" ]; then
    echo "Error: --image is required"
    show_help
    exit 1
fi

if [ ! -f "$CONFIG" ]; then
    echo "Error: Config file not found: $CONFIG"
    exit 1
fi

if [ "$HOST" != "local" ] && [ -z "$SSH_KEY" ]; then
    echo "Error: --ssh-key is required for remote deployment"
    exit 1
fi

echo "════════════════════════════════════════════════"
echo "  Deploying Photo Sync Backend"
echo "════════════════════════════════════════════════"
echo ""
echo "Configuration:"
echo "  Host: $HOST"
echo "  Port: $PORT"
echo "  Image: $IMAGE"
echo "  Config: $CONFIG"
if [ -n "$SSH_KEY" ]; then
    echo "  SSH Key: $SSH_KEY"
fi
echo ""

if [ "$HOST" = "local" ]; then
    echo "Deploying locally with docker-compose..."
    echo ""
    
    if ! command -v docker-compose &> /dev/null; then
        echo "Error: docker-compose not found. Please install docker-compose."
        exit 1
    fi
    
    echo "Checking if image exists locally..."
    if ! docker image inspect "$IMAGE" &> /dev/null; then
        echo "Error: Docker image not found: $IMAGE"
        echo "Build the image first with: ./scripts/build.sh"
        exit 1
    fi
    echo "✓ Image found"
    echo ""
    
    export PHOTO_SYNC_IMAGE="$IMAGE"
    export PHOTO_SYNC_PORT="$PORT"
    
    echo "Stopping existing containers..."
    docker-compose down 2>/dev/null || true
    echo ""
    
    echo "Starting container..."
    docker-compose up -d
    echo ""
    
    echo "Waiting for service to start..."
    sleep 3
    echo ""
    
    echo "Checking service status..."
    if docker-compose ps | grep -q "Up"; then
        echo "✓ Service is running"
    else
        echo "Error: Service failed to start"
        echo "Check logs with: docker-compose logs"
        exit 1
    fi
    
else
    echo "Deploying to remote host: $HOST"
    echo ""
    
    if [ ! -f "$SSH_KEY" ]; then
        echo "Error: SSH key not found: $SSH_KEY"
        exit 1
    fi
    
    echo "Testing SSH connection..."
    if ! ssh -i "$SSH_KEY" -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@"$HOST" "echo 'SSH OK'" &> /dev/null; then
        echo "Error: Cannot connect to $HOST via SSH"
        exit 1
    fi
    echo "✓ SSH connection successful"
    echo ""
    
    echo "Copying configuration to remote host..."
    ssh -i "$SSH_KEY" root@"$HOST" "mkdir -p /opt/photo-sync-backend"
    scp -i "$SSH_KEY" "$CONFIG" root@"$HOST":/opt/photo-sync-backend/config.yaml
    scp -i "$SSH_KEY" docker-compose.yml root@"$HOST":/opt/photo-sync-backend/
    echo "✓ Configuration copied"
    echo ""
    
    echo "Deploying on remote host..."
    ssh -i "$SSH_KEY" root@"$HOST" "cd /opt/photo-sync-backend && \
        export PHOTO_SYNC_IMAGE='$IMAGE' && \
        export PHOTO_SYNC_PORT='$PORT' && \
        docker-compose pull && \
        docker-compose down 2>/dev/null || true && \
        docker-compose up -d"
    echo ""
    
    echo "Waiting for service to start..."
    sleep 5
    echo ""
    
    echo "Checking service status..."
    ssh -i "$SSH_KEY" root@"$HOST" "cd /opt/photo-sync-backend && docker-compose ps"
    echo ""
fi

echo "════════════════════════════════════════════════"
echo "  Deployment Complete!"
echo "════════════════════════════════════════════════"
echo ""

if [ "$HOST" = "local" ]; then
    echo "Service URL: https://localhost:$PORT"
else
    echo "Service URL: https://$HOST:$PORT"
fi

echo ""
echo "Test the service:"
if [ "$HOST" = "local" ]; then
    echo "  curl -k https://localhost:$PORT/api/auth/login \\"
else
    echo "  curl -k https://$HOST:$PORT/api/auth/login \\"
fi
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"test\",\"password\":\"test\"}'"
echo ""
echo "View logs:"
if [ "$HOST" = "local" ]; then
    echo "  docker-compose logs -f"
else
    echo "  ssh -i $SSH_KEY root@$HOST 'cd /opt/photo-sync-backend && docker-compose logs -f'"
fi
echo ""

