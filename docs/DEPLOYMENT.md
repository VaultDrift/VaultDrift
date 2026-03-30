# Deployment Guide

## Docker (Single Container)

### Quick Start

```bash
# Pull and run
docker run -d \
  --name vaultdrift \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  vaultdrift/vaultdrift:latest
```

### With Custom Config

```bash
docker run -d \
  --name vaultdrift \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  vaultdrift/vaultdrift:latest
```

## Docker Compose

### Basic Setup

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f vaultdrift

# Stop services
docker-compose down
```

### With S3/MinIO Backend

```bash
# Start with MinIO
docker-compose --profile s3 up -d

# Access MinIO console at http://localhost:9001
# Default credentials: minioadmin / minioadmin
```

## Kubernetes

### Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Ingress controller (nginx recommended)
- cert-manager (for TLS)

### Deploy

```bash
# Create namespace
kubectl create namespace vaultdrift

# Apply deployment
kubectl apply -f deploy/k8s/ -n vaultdrift

# Check status
kubectl get pods -n vaultdrift
kubectl get svc -n vaultdrift
```

### Update

```bash
# Rolling update
kubectl set image deployment/vaultdrift vaultdrift=vaultdrift/vaultdrift:v1.1.0 -n vaultdrift

# Check rollout status
kubectl rollout status deployment/vaultdrift -n vaultdrift
```

## Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name vaultdrift.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    client_max_body_size 100M;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VAULTDRIFT_SERVER_PORT` | HTTP server port | `8080` |
| `VAULTDRIFT_SERVER_HOST` | Server bind address | `0.0.0.0` |
| `VAULTDRIFT_DATABASE_PATH` | Database file path | `./vaultdrift.db` |
| `VAULTDRIFT_STORAGE_BACKEND` | Storage backend (`local` or `s3`) | `local` |
| `VAULTDRIFT_STORAGE_LOCAL_PATH` | Local storage path | `./storage` |
| `VAULTDRIFT_STORAGE_S3_ENDPOINT` | S3 endpoint | - |
| `VAULTDRIFT_STORAGE_S3_BUCKET` | S3 bucket name | - |
| `VAULTDRIFT_STORAGE_S3_REGION` | S3 region | - |
| `VAULTDRIFT_STORAGE_S3_ACCESS_KEY` | S3 access key | - |
| `VAULTDRIFT_STORAGE_S3_SECRET_KEY` | S3 secret key | - |
| `VAULTDRIFT_AUTH_JWT_SECRET` | JWT signing secret | (random) |
| `VAULTDRIFT_LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |

## Backup

### Database Backup

```bash
# Docker
docker exec vaultdrift sqlite3 /data/vaultdrift.db ".backup /data/backup.db"
docker cp vaultdrift:/data/backup.db ./backup-$(date +%Y%m%d).db

# Kubernetes
kubectl cp vaultdrift-xxx:/data/vaultdrift.db ./backup.db -n vaultdrift
```

### Automated Backups

```bash
# Create backup script
cat > backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backups/vaultdrift"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup database
docker exec vaultdrift sqlite3 /data/vaultdrift.db ".backup /tmp/backup.db"
docker cp vaultdrift:/tmp/backup.db "$BACKUP_DIR/db-$DATE.db"

# Backup storage
tar czf "$BACKUP_DIR/storage-$DATE.tar.gz" -C /var/lib/vaultdrift storage

# Keep only last 7 days
find $BACKUP_DIR -name "*.db" -mtime +7 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete
EOF
chmod +x backup.sh

# Add to crontab (daily at 2 AM)
0 2 * * * /path/to/backup.sh
```

## Security Checklist

- [ ] Use HTTPS/TLS in production
- [ ] Set strong JWT secret
- [ ] Enable TOTP for admin users
- [ ] Configure firewall rules
- [ ] Set up automated backups
- [ ] Use non-root user in containers
- [ ] Enable audit logging
- [ ] Configure rate limiting
