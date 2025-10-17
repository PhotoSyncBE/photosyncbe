# Photo Sync Backend

Secure, high-performance Go backend providing HTTPS REST API for photo synchronization with pluggable authentication and storage backends.

## Features

- Multiple authentication methods: Active Directory, LDAP, Local Users, OAuth2
- Multiple storage backends: SMB/CIFS, S3-compatible, NFS
- JWT token-based API authentication with encrypted credentials
- Connection pooling for performance
- TLS/HTTPS with configurable certificates
- Per-user directory isolation
- Docker, Kubernetes, and AWS ECS deployment ready

## Architecture

```
┌─────────────────────────────────────────────┐
│           Photo Sync Backend                │
│  ┌────────────────────────────────────────┐ │
│  │  REST API (chi router)                 │ │
│  ├────────────────────────────────────────┤ │
│  │  JWT Middleware                        │ │
│  ├────────────────────────────────────────┤ │
│  │  Authentication (Interface)            │ │
│  │  ├─ Active Directory                   │ │
│  │  ├─ Generic LDAP                       │ │
│  │  ├─ Local Users                        │ │
│  │  └─ OAuth2                             │ │
│  ├────────────────────────────────────────┤ │
│  │  Storage Backend (Interface)           │ │
│  │  ├─ SMB/CIFS                           │ │
│  │  ├─ S3-Compatible                      │ │
│  │  └─ NFS                                │ │
│  ├────────────────────────────────────────┤ │
│  │  Connection Pool (per-user, TTL-based) │ │
│  └────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

## Supported Combinations

**Any authentication method works with any storage backend.** Common configurations:

| Authentication | Storage | Use Case |
|---------------|---------|----------|
| Active Directory | SMB | Enterprise Windows environment |
| Active Directory | S3 | Hybrid cloud with AD |
| LDAP | NFS | Linux/Unix environment |
| LDAP | SMB | Linux LDAP auth + Windows file server |
| Local Users | SMB | Small office/home with existing SMB share |
| Local Users | S3 | Simple cloud deployment |
| Local Users | NFS | Simple deployment with NFS storage |
| OAuth2 | S3 | Public-facing application |
| OAuth2 | Local | Development/testing |

## Quick Start

### 1. Choose Your Configuration

Copy an example config from `examples/`:
```bash
cp examples/ad-smb.yaml config.yaml
# OR
cp examples/local-s3.yaml config.yaml
# OR
cp examples/ldap-nfs.yaml config.yaml
```

### 2. Configure Your Backend

Edit `config.yaml` and replace all `CHANGE_ME` placeholders with your actual values.

Generate JWT keys:
```bash
openssl rand -base64 64  # For secret_key
openssl rand -base64 32 | cut -c1-32  # For encryption_key (must be 32 chars)
```

### 3. Build and Run

**Docker:**
```bash
docker build -t photo-sync-backend .
docker run -d -p 8443:8443 -v $(pwd)/config.yaml:/root/config.yaml photo-sync-backend
```

**Local:**
```bash
go build -o photo-sync-backend ./cmd/server
./photo-sync-backend
```

## Authentication Methods

### Active Directory

```yaml
auth:
  type: "active_directory"

ldap:
  server: "ldap://YOUR_AD_IP:389"
  base_dn: "DC=example,DC=com"
  user_filter: "(sAMAccountName={username})"
  bind_dn: "CN=service,OU=Users,DC=example,DC=com"
  bind_pass: "YOUR_PASSWORD"
```

Best for: Enterprise environments with Windows AD

### Generic LDAP

```yaml
auth:
  type: "ldap"

ldap_generic:
  server: "ldap://YOUR_LDAP_IP:389"
  base_dn: "dc=example,dc=com"
  user_filter: "(uid={username})"
  bind_dn: "cn=readonly,dc=example,dc=com"
  bind_pass: "YOUR_PASSWORD"
  user_dn_pattern: "uid={username},ou=people,dc=example,dc=com"
```

Best for: OpenLDAP, FreeIPA, 389 Directory Server

### Local Users

```yaml
auth:
  type: "local"

local_auth:
  users_file: "/app/users.json"
```

users.json format:
```json
[
  {
    "username": "user1",
    "password_hash": "$2a$10$...",
    "email": "user1@example.com",
    "full_name": "User One"
  }
]
```

Generate password hash:
```bash
htpasswd -bnBC 10 "" password | tr -d ':\n'
```

Best for: Small deployments, testing, no external auth system

### OAuth2

```yaml
auth:
  type: "oauth2"

oauth2:
  provider: "google"
  client_id: "YOUR_CLIENT_ID"
  client_secret: "YOUR_CLIENT_SECRET"
  redirect_url: "https://yourdomain.com:8443/auth/callback"
```

Supported providers: google, github

Best for: Public-facing apps, social login

## Storage Backends

### SMB/CIFS

```yaml
storage:
  type: "smb"

smb:
  server: "192.168.1.100"
  port: 445
  share: "Photos"
  path: ""
  domain: "WORKGROUP"
```

Directory structure: `//server/share/{username}/`

Best for: Windows file servers, TrueNAS, Samba

### S3-Compatible

```yaml
storage:
  type: "s3"

s3:
  endpoint: "https://s3.amazonaws.com"
  region: "us-east-1"
  bucket: "my-photos"
  access_key: "YOUR_ACCESS_KEY"
  secret_key: "YOUR_SECRET_KEY"
  path_prefix: "photos"
  use_ssl: true
```

Object key format: `{bucket}/{path_prefix}/{username}/{filename}`

Supports: AWS S3, MinIO, Wasabi, DigitalOcean Spaces, Backblaze B2

Best for: Cloud storage, scalability, geographic distribution

### NFS

```yaml
storage:
  type: "nfs"

nfs:
  server: "192.168.1.100"
  export: "/export/photos"
  path: ""
```

Directory structure: `{export}/{path}/{username}/`

Best for: Linux/Unix file servers, existing NFS infrastructure

## API Endpoints

### Authentication

```
POST /api/auth/login
Body: {"username": "user", "password": "pass"}
Response: {"token": "eyJ...", "expires_at": 1234567890}
```

### Photo Operations

All require `Authorization: Bearer <token>` header

```
GET    /api/photos              List user's photos
POST   /api/photos              Upload photo (multipart/form-data)
GET    /api/photos/{filename}   Download photo
DELETE /api/photos/{filename}   Delete photo
GET    /api/photos/{filename}/info  Get photo metadata
```

## Building

### Build with Existing Certificates

```bash
./scripts/build.sh \
  --cert-path /etc/letsencrypt/live/yourdomain.com \
  --domain yourdomain.com \
  --image-tag photo-sync-backend:v1.0
```

**Note:** If your `config.yaml` contains placeholder JWT keys (like `"CHANGE_ME"`), the build script will automatically generate secure keys for you.

### Build with Self-Signed Certificates (Development)

```bash
./scripts/generate-certs.sh --domain localhost
./scripts/build.sh \
  --cert-path ./certs \
  --domain localhost \
  --image-tag photo-sync-backend:dev
```

### Build and Push to Registry

```bash
./scripts/build.sh \
  --cert-path ./certs \
  --domain yourdomain.com \
  --image-tag myregistry.com/photo-sync:v1.0 \
  --registry myregistry.com
```

## Deployment

### Local Deployment

```bash
./scripts/deploy.sh \
  --host local \
  --image photo-sync-backend:v1.0 \
  --config config.yaml
```

### Remote Deployment

```bash
./scripts/deploy.sh \
  --host 192.168.1.100 \
  --ssh-key ~/.ssh/id_rsa \
  --image myregistry.com/photo-sync:v1.0 \
  --config config.yaml
```

### Custom Port

```bash
./scripts/deploy.sh \
  --host local \
  --port 9443 \
  --image photo-sync-backend:latest
```

### Kubernetes

```bash
kubectl apply -f examples/kubernetes/deployment.yaml
kubectl expose deployment photo-sync-backend --type=LoadBalancer --port=8443
```

### AWS ECS/Fargate

1. Build and push image to ECR
2. Create ECS task definition using `examples/aws/ecs-task-definition.json`
3. Create ECS service
4. Configure Application Load Balancer for HTTPS

Full AWS deployment:
```bash
aws ecr create-repository --repository-name photo-sync-backend
docker build -t photo-sync-backend .
docker tag photo-sync-backend:latest $ECR_REPO:latest
docker push $ECR_REPO:latest

aws ecs register-task-definition --cli-input-json file://examples/aws/ecs-task-definition.json
aws ecs create-service --cluster my-cluster --service-name photo-sync --task-definition photo-sync-backend
```

## Certificate Management

### Obtaining Certificates

#### Option 1: Let's Encrypt (Recommended for Production)

```bash
# Install certbot
sudo apt-get install certbot  # Debian/Ubuntu
sudo yum install certbot       # RHEL/CentOS

# Get certificate
sudo certbot certonly --standalone -d yourdomain.com

# Certificates will be in:
# /etc/letsencrypt/live/yourdomain.com/
```

#### Option 2: Self-Signed (Development Only)

```bash
./scripts/generate-certs.sh --domain localhost
```

### Renewing Certificates

Certificates expire after 90 days. To renew:

```bash
# Renew with certbot
sudo certbot renew

# Rebuild and redeploy with new certificates
./scripts/build.sh \
  --cert-path /etc/letsencrypt/live/yourdomain.com \
  --domain yourdomain.com \
  --image-tag photo-sync-backend:v1.1

./scripts/deploy.sh \
  --host your-server \
  --ssh-key ~/.ssh/id_rsa \
  --image photo-sync-backend:v1.1
```

## JWT Key Management

### Automatic Key Generation

The build script automatically generates secure JWT keys when:
- Keys are missing from `config.yaml`
- Keys contain placeholders like `"CHANGE_ME"` or `"PLACEHOLDER"`
- Keys are too short (secret key < 32 chars, encryption key ≠ 32 chars)

Generated keys are:
- **Secret Key**: 64 bytes base64-encoded (signs JWT tokens)
- **Encryption Key**: 32 characters (encrypts user passwords in tokens)

### Manual Key Generation

If you prefer to generate keys manually:

```bash
# Generate secret key (64 bytes)
openssl rand -base64 64

# Generate encryption key (exactly 32 chars)
openssl rand -base64 32 | cut -c1-32
```

### Key Security

- **Save keys securely** when auto-generated
- **Use same keys** across deployments to avoid user logout
- **Rotate keys periodically** for security (invalidates old tokens)
- **Backup keys** - losing them requires all users to re-authenticate

### Key Rotation

To rotate keys (logs out all users):

```bash
# Edit config.yaml with new keys, or
# Delete/change existing JWT key values to trigger auto-generation

./scripts/build.sh --cert-path ./certs --domain yourdomain.com
./scripts/deploy.sh --host your-server --image photo-sync-backend:new-version
```

## Configuration

### Setup

1. Copy the example configuration:
   ```bash
   cp config.example.yaml config.yaml
   ```

2. Edit `config.yaml` with your environment settings:
   - Choose your authentication type (`auth.type`)
   - Choose your storage backend (`storage.type`)
   - Configure the corresponding authentication and storage sections
   - JWT keys will be auto-generated during build if left as default values

**Important:** `config.yaml` contains sensitive information and is automatically ignored by Git. Never commit it to version control.

### Authentication & Storage Options

See the `examples/` directory for complete configuration examples for different combinations:
- `ad-smb.yaml` - Active Directory + SMB
- `ldap-nfs.yaml` - LDAP + NFS
- `local-smb.yaml` - Local users + SMB
- `local-s3.yaml` - Local users + S3
- `oauth2-s3.yaml` - OAuth2 + S3

## Configuration Reference

### Server

```yaml
server:
  host: "0.0.0.0"
  port: "8443"
  tls:
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
```

### JWT

```yaml
jwt:
  secret_key: "CHANGE_ME"
  encryption_key: "CHANGE_ME_32_CHARS"
  issuer: "photo-sync-backend"
  expiry: "24h"
```

### Connection Pool

```yaml
pool:
  connection_ttl: "10m"
```

## Security

- Passwords encrypted in JWT using AES-256-GCM
- Per-user directory isolation
- TLS 1.2+ required
- No default credentials (validation enforces configuration)
- Configurable JWT expiry
- Connection pool TTL forces re-authentication

## Troubleshooting

### Authentication Fails

**Active Directory:**
```bash
ldapsearch -x -H ldap://AD_IP:389 -b "DC=domain,DC=com"
```

**Local Auth:**
Verify users.json exists and password hashes are correct

**OAuth2:**
Verify client_id and client_secret are correct

### Storage Connection Fails

**SMB:**
```bash
smbclient //SERVER/SHARE -U username
```

**S3:**
```bash
aws s3 ls s3://bucket-name --endpoint-url https://endpoint
```

**NFS:**
```bash
showmount -e NFS_SERVER
```

### Config Validation Errors

Ensure all `CHANGE_ME` placeholders are replaced with actual values. The backend will refuse to start with placeholder values.

## Development

```bash
go mod download
go build -o photo-sync-backend ./cmd/server
./photo-sync-backend
```

## Testing

```bash
# Login
curl -k https://localhost:8443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user","password":"pass"}'

# Upload
TOKEN="your_jwt_token"
curl -k -X POST https://localhost:8443/api/photos \
  -H "Authorization: Bearer $TOKEN" \
  -F "photo=@test.jpg"

# List
curl -k https://localhost:8443/api/photos \
  -H "Authorization: Bearer $TOKEN"
```

## Examples

See `examples/` directory for:
- Complete configuration files for all auth/storage combinations
- Docker Compose configurations
- Kubernetes manifests
- AWS ECS task definitions


