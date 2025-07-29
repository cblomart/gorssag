# Docker Container Usage Guide

## 🐳 GitHub Packages Container Registry

The RSS Aggregator is available as a Docker container from GitHub Packages.

### **Container Location**
```
ghcr.io/cblomart/gorssag
```

### **Available Tags**

#### **Latest Versions**
- `latest` - Latest stable build from main branch
- `v1.0.1` - Latest release version
- `v1.0.0` - Initial release version

#### **Development Versions**
- `main` - Latest build from main branch
- `sha-{commit}` - Build from specific commit

### **Quick Start**

#### **Pull and Run**
```bash
# Pull the latest version
docker pull ghcr.io/cblomart/gorssag:latest

# Run with default configuration
docker run -p 8080:8080 ghcr.io/cblomart/gorssag:latest
```

#### **Using Docker Compose**
```bash
# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  rss-aggregator:
    image: ghcr.io/cblomart/gorssag:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - CACHE_TTL=30m
      - POLL_INTERVAL=30m
      - ENABLE_SPA=true
      - ENABLE_SWAGGER=true
      - FEED_TOPIC_TECH=https://feeds.feedburner.com/TechCrunch|AI,blockchain
      - FEED_TOPIC_NEWS=https://feeds.bbci.co.uk/news/rss.xml|breaking,update
      - FEED_TOPIC_PROGRAMMING=https://feeds.feedburner.com/TheHackersNews|security,cyber
    volumes:
      - rss_data:/app/data
    restart: unless-stopped

volumes:
  rss_data:
EOF

# Start the service
docker-compose up -d
```

### **Configuration Options**

#### **Environment Variables**
```bash
# Basic Configuration
PORT=8080                           # Server port
DATA_DIR=/app/data                  # Data storage directory
CACHE_TTL=30m                       # Cache time-to-live
POLL_INTERVAL=30m                   # Background polling interval

# RSS Feed Configuration
FEED_TOPIC_TECH=https://feeds.feedburner.com/TechCrunch|AI,blockchain
FEED_TOPIC_NEWS=https://feeds.bbci.co.uk/news/rss.xml|breaking,update
FEED_TOPIC_PROGRAMMING=https://feeds.feedburner.com/TheHackersNews|security,cyber

# Web Interface Configuration
ENABLE_SPA=true                     # Enable Single Page Application
ENABLE_SWAGGER=true                 # Enable Swagger documentation

# Security Configuration
ENABLE_RATE_LIMIT=true              # Enable rate limiting
RATE_LIMIT_PER_SECOND=10.0          # Requests per second per IP
RATE_LIMIT_BURST=20                 # Burst limit
ENABLE_CORS=true                    # Enable CORS
ALLOWED_ORIGINS=*                   # Allowed CORS origins
ENABLE_SECURITY_HEADERS=true        # Enable security headers
MAX_REQUEST_SIZE=10485760           # Max request size (10MB)
ENABLE_REQUEST_ID=true              # Enable request ID tracking
```

#### **Volume Mounts**
```bash
# Persistent data storage
-v /path/to/data:/app/data

# Custom configuration (if needed)
-v /path/to/config:/app/config
```

### **Usage Examples**

#### **Basic Usage**
```bash
# Simple run
docker run -d \
  --name rss-aggregator \
  -p 8080:8080 \
  ghcr.io/cblomart/gorssag:latest
```

#### **With Custom Configuration**
```bash
# Run with custom feeds and settings
docker run -d \
  --name rss-aggregator \
  -p 8080:8080 \
  -e FEED_TOPIC_TECH="https://feeds.feedburner.com/TechCrunch|AI,blockchain" \
  -e FEED_TOPIC_NEWS="https://feeds.bbci.co.uk/news/rss.xml|breaking,update" \
  -e ENABLE_SPA=true \
  -e ENABLE_SWAGGER=true \
  -e ENABLE_RATE_LIMIT=true \
  -v rss_data:/app/data \
  ghcr.io/cblomart/gorssag:v1.0.1
```

#### **Production Deployment**
```bash
# Production-ready deployment
docker run -d \
  --name rss-aggregator \
  --restart unless-stopped \
  -p 8080:8080 \
  -e PORT=8080 \
  -e CACHE_TTL=30m \
  -e POLL_INTERVAL=30m \
  -e ENABLE_RATE_LIMIT=true \
  -e RATE_LIMIT_PER_SECOND=10.0 \
  -e ENABLE_SECURITY_HEADERS=true \
  -e ENABLE_CORS=true \
  -e ALLOWED_ORIGINS="https://yourdomain.com" \
  -e FEED_TOPIC_TECH="https://feeds.feedburner.com/TechCrunch|AI,blockchain" \
  -e FEED_TOPIC_NEWS="https://feeds.bbci.co.uk/news/rss.xml|breaking,update" \
  -v /opt/rss-aggregator/data:/app/data \
  ghcr.io/cblomart/gorssag:latest
```

### **Accessing the Application**

#### **Web Interfaces**
- **SPA Interface**: http://localhost:8080/
- **Swagger Documentation**: http://localhost:8080/swagger/

#### **API Endpoints**
- **Health Check**: http://localhost:8080/health
- **Available Topics**: http://localhost:8080/api/v1/topics
- **Feed Data**: http://localhost:8080/api/v1/feeds/{topic}
- **Feed Info**: http://localhost:8080/api/v1/feeds/{topic}/info
- **Poller Status**: http://localhost:8080/api/v1/poller/status

### **Monitoring and Logs**

#### **Container Logs**
```bash
# View logs
docker logs rss-aggregator

# Follow logs
docker logs -f rss-aggregator

# View recent logs
docker logs --tail 100 rss-aggregator
```

#### **Health Monitoring**
```bash
# Health check
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0.1",
  "poller_active": true
}
```

### **Troubleshooting**

#### **Common Issues**

**Container won't start:**
```bash
# Check logs
docker logs rss-aggregator

# Check if port is already in use
netstat -tulpn | grep 8080
```

**Permission issues with data directory:**
```bash
# Fix permissions
sudo chown -R 1000:1000 /path/to/data
```

**Memory issues:**
```bash
# Check container resource usage
docker stats rss-aggregator

# Increase memory limit if needed
docker run --memory=512m ghcr.io/cblomart/gorssag:latest
```

#### **Debug Mode**
```bash
# Run with debug logging
docker run -e LOG_LEVEL=debug ghcr.io/cblomart/gorssag:latest
```

### **Security Considerations**

#### **Network Security**
```bash
# Run on internal network only
docker run --network internal ghcr.io/cblomart/gorssag:latest

# Use reverse proxy
docker run -p 127.0.0.1:8080:8080 ghcr.io/cblomart/gorssag:latest
```

#### **Data Security**
```bash
# Encrypt data volume
docker run -v encrypted_data:/app/data ghcr.io/cblomart/gorssag:latest

# Use read-only root filesystem
docker run --read-only ghcr.io/cblomart/gorssag:latest
```

### **Performance Tuning**

#### **Resource Limits**
```bash
# Set CPU and memory limits
docker run \
  --cpus=1.0 \
  --memory=512m \
  --memory-swap=1g \
  ghcr.io/cblomart/gorssag:latest
```

#### **Optimization Tips**
- **Use volume mounts** for persistent data
- **Set appropriate cache TTL** based on your needs
- **Configure polling intervals** to balance freshness vs. load
- **Enable rate limiting** for production use
- **Use reverse proxy** for SSL termination

### **Updates and Maintenance**

#### **Updating the Container**
```bash
# Pull latest version
docker pull ghcr.io/cblomart/gorssag:latest

# Stop and remove old container
docker stop rss-aggregator
docker rm rss-aggregator

# Run new version
docker run -d --name rss-aggregator -p 8080:8080 ghcr.io/cblomart/gorssag:latest
```

#### **Backup and Restore**
```bash
# Backup data
docker run --rm -v rss_data:/data -v $(pwd):/backup alpine tar czf /backup/rss-data-backup.tar.gz -C /data .

# Restore data
docker run --rm -v rss_data:/data -v $(pwd):/backup alpine tar xzf /backup/rss-data-backup.tar.gz -C /data
```

### **Integration Examples**

#### **With Nginx Reverse Proxy**
```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### **With Docker Swarm**
```yaml
version: '3.8'
services:
  rss-aggregator:
    image: ghcr.io/cblomart/gorssag:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ENABLE_SPA=true
    volumes:
      - rss_data:/app/data
    deploy:
      replicas: 2
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure

volumes:
  rss_data:
```

---

**For more information, visit:**
- [GitHub Repository](https://github.com/cblomart/gorssag)
- [API Documentation](API.md)
- [Security Guide](SECURITY.md) 