# Quick Start Guide

Get Photo Sync Backend running in 4 steps.

## Prerequisites

- Docker
- TLS certificates (or generate self-signed)
- Go 1.24+ (only if building from source)

## Step 1: Prepare Certificates

### Self-Signed (Development)
```bash
./scripts/generate-certs.sh --domain localhost
```

### Let's Encrypt (Production)
```bash
sudo certbot certonly --standalone -d photos.yourdomain.com
```

## Step 2: Choose Configuration

Copy an example and edit with your settings:

```bash
cp examples/ad-smb.yaml config.yaml
# Edit config.yaml with your LDAP/SMB settings
# JWT keys will be auto-generated during build if needed
```

## Step 3: Build Docker Image

### With Self-Signed Certificates
```bash
./scripts/build.sh \
  --cert-path ./certs \
  --domain localhost \
  --image-tag photo-sync-backend:latest
```

### With Let's Encrypt Certificates
```bash
./scripts/build.sh \
  --cert-path /etc/letsencrypt/live/photos.yourdomain.com \
  --domain photos.yourdomain.com \
  --image-tag photo-sync-backend:latest
```

## Step 4: Deploy

### Local Deployment
```bash
./scripts/deploy.sh \
  --host local \
  --image photo-sync-backend:latest \
  --config config.yaml
```

### Remote Deployment
```bash
./scripts/deploy.sh \
  --host 192.168.1.100 \
  --ssh-key ~/.ssh/id_rsa \
  --image photo-sync-backend:latest \
  --config config.yaml
```

## Testing

### 1. Test Login
```bash
curl -k https://localhost:8443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"YOUR_USER","password":"YOUR_PASS"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJI...",
  "expires_at": 1234567890
}
```

### 2. Test Upload
```bash
TOKEN="your_token_from_above"

curl -k -X POST https://localhost:8443/api/photos \
  -H "Authorization: Bearer $TOKEN" \
  -F "photo=@test.jpg"
```

### 3. List Photos
```bash
curl -k https://localhost:8443/api/photos \
  -H "Authorization: Bearer $TOKEN"
```

## Common Issues

### "jwt secret_key must be set (no placeholders allowed)"

Generate new JWT keys:
```bash
openssl rand -base64 64  # secret_key
openssl rand -base64 32 | cut -c1-32  # encryption_key
```

### "failed to connect to LDAP/SMB/NFS"

Verify server is reachable:
```bash
ping SERVER_IP
telnet SERVER_IP PORT
```

### "invalid credentials"

For Active Directory/LDAP: Test credentials manually
For Local Auth: Verify password hash is correct

## Configuration Examples

All examples available in `examples/` directory:
- `ad-smb.yaml` - Active Directory + SMB
- `ldap-nfs.yaml` - LDAP + NFS
- `local-smb.yaml` - Local Users + SMB
- `local-s3.yaml` - Local Users + S3
- `oauth2-s3.yaml` - OAuth2 + S3

## Next Steps

- Configure mobile app to use `https://your-server:8443`
- Set up automatic certificate renewal
- Configure monitoring and logging
- Review security settings

For detailed information, see [README.md](README.md)
