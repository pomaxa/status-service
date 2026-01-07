# Production Deployment Guide

This guide covers deploying Status Incident Service in production environments.

## Quick Start (Docker)

The fastest way to deploy:

```bash
# Clone and run
git clone https://github.com/pomaxa/status-service.git
cd status-service
docker-compose up -d
```

Service will be available at `http://localhost:8080`

---

## Deployment Options

### Option 1: Docker Compose (Recommended)

Best for: Most deployments, easy management, automatic restarts.

```bash
# Start service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop service
docker-compose down

# Update to latest version
docker-compose pull
docker-compose up -d
```

**Custom configuration** - create `docker-compose.override.yml`:

```yaml
version: '3.8'
services:
  status-incident:
    ports:
      - "127.0.0.1:8080:8080"  # Only localhost (use with reverse proxy)
    environment:
      - TZ=Europe/Moscow
    command: ["./status-incident", "-heartbeat", "30s"]
```

### Option 2: Systemd Service (Bare Metal)

Best for: VPS, dedicated servers, maximum control.

1. **Build binary:**
```bash
CGO_ENABLED=1 go build -o status-incident .
```

2. **Create user and directories:**
```bash
sudo useradd -r -s /bin/false status-incident
sudo mkdir -p /opt/status-incident/{data,templates,static}
sudo cp status-incident /opt/status-incident/
sudo cp -r templates/* /opt/status-incident/templates/
sudo cp -r static/* /opt/status-incident/static/
sudo chown -R status-incident:status-incident /opt/status-incident
```

3. **Install systemd unit:**
```bash
sudo cp deploy/status-incident.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable status-incident
sudo systemctl start status-incident
```

4. **Check status:**
```bash
sudo systemctl status status-incident
sudo journalctl -u status-incident -f
```

### Option 3: Docker (Manual)

```bash
# Build image
docker build -t status-incident .

# Run container
docker run -d \
  --name status-incident \
  --restart unless-stopped \
  -p 8080:8080 \
  -v status-data:/app/data \
  status-incident
```

---

## Reverse Proxy Setup (nginx)

**Required for production** - provides HTTPS, caching, and security.

1. **Install nginx:**
```bash
sudo apt install nginx
```

2. **Copy configuration:**
```bash
sudo cp deploy/nginx.conf /etc/nginx/sites-available/status-incident
sudo ln -s /etc/nginx/sites-available/status-incident /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

3. **Setup SSL with Let's Encrypt:**
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d status.example.com
```

---

## Configuration Reference

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | HTTP server address |
| `-db` | `status.db` | SQLite database path |
| `-templates` | `templates` | Templates directory |
| `-heartbeat` | `60s` | Heartbeat check interval |

### Examples

```bash
# Custom port
./status-incident -addr :3000

# Custom database location
./status-incident -db /var/lib/status-incident/data.db

# Faster heartbeat checks
./status-incident -heartbeat 30s

# All options
./status-incident \
  -addr :8080 \
  -db /app/data/status.db \
  -templates /app/templates \
  -heartbeat 60s
```

### Docker Environment Variables

| Variable | Description |
|----------|-------------|
| `TZ` | Timezone (e.g., `Europe/Moscow`, `UTC`) |

---

## Backup & Restore

### Automatic Backup (Cron)

1. **Install backup script:**
```bash
sudo cp deploy/backup.sh /opt/status-incident/
sudo chmod +x /opt/status-incident/backup.sh
```

2. **Add to crontab:**
```bash
# Daily backup at 2:00 AM
echo "0 2 * * * /opt/status-incident/backup.sh" | sudo crontab -u status-incident -
```

### Manual Backup

**Docker:**
```bash
# Export via API (recommended)
curl http://localhost:8080/api/export > backup-$(date +%Y%m%d).json

# Copy SQLite file
docker cp status-incident:/app/data/status.db ./backup-$(date +%Y%m%d).db
```

**Systemd:**
```bash
# Export via API
curl http://localhost:8080/api/export > backup-$(date +%Y%m%d).json

# Copy SQLite file (stop service first for consistency)
sudo systemctl stop status-incident
sudo cp /opt/status-incident/data/status.db ./backup-$(date +%Y%m%d).db
sudo systemctl start status-incident
```

### Restore from Backup

**From JSON export:**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d @backup-20240115.json \
  http://localhost:8080/api/import
```

**From SQLite file:**
```bash
# Docker
docker-compose down
docker run --rm -v status-data:/data -v $(pwd):/backup alpine \
  cp /backup/backup-20240115.db /data/status.db
docker-compose up -d

# Systemd
sudo systemctl stop status-incident
sudo cp backup-20240115.db /opt/status-incident/data/status.db
sudo chown status-incident:status-incident /opt/status-incident/data/status.db
sudo systemctl start status-incident
```

---

## Security Checklist

### Essential

- [ ] **HTTPS only** - Use nginx with SSL certificate
- [ ] **Firewall** - Only expose ports 80/443
- [ ] **Internal network** - Bind app to localhost, proxy via nginx

### Recommended

- [ ] **Basic Auth for /admin** - Add nginx authentication
- [ ] **Rate limiting** - Configure in nginx
- [ ] **Regular backups** - Setup automated backups
- [ ] **Log rotation** - Configure logrotate for systemd

### Nginx Basic Auth for Admin

```bash
# Create password file
sudo htpasswd -c /etc/nginx/.htpasswd admin

# Add to nginx location block (see deploy/nginx.conf)
```

### Firewall (UFW)

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

---

## Monitoring

### Health Check Endpoint

```bash
# Simple check
curl -f http://localhost:8080/ || echo "Service down"

# With timeout
curl -f --connect-timeout 5 http://localhost:8080/api/systems || exit 1
```

### Docker Health Check

Add to `docker-compose.override.yml`:
```yaml
services:
  status-incident:
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Uptime Monitoring

External services to monitor your status page:
- UptimeRobot (free)
- Pingdom
- StatusCake

---

## Troubleshooting

### Service won't start

```bash
# Check logs
docker-compose logs status-incident
# or
sudo journalctl -u status-incident -n 100

# Common issues:
# - Port already in use: change -addr flag
# - Database permissions: check file ownership
# - Missing templates: verify templates directory exists
```

### Database locked

SQLite can lock if accessed by multiple processes:
```bash
# Check for locks
fuser /opt/status-incident/data/status.db

# Restart service
sudo systemctl restart status-incident
```

### High memory usage

SQLite WAL files can grow large:
```bash
# Checkpoint WAL manually
sqlite3 /opt/status-incident/data/status.db "PRAGMA wal_checkpoint(TRUNCATE);"
```
