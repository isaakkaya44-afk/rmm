# Hetzner CPX32 Deployment Guide

## Prerequisites
- Hetzner CPX32 VPS (Ubuntu 22.04/24.04 LTS)
- Domain pointing to server IP
- SSH access

## Quick Start

```bash
# 1. Server'da Docker kur
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Çıkış yap / giriş yap

# 2. Projeyi clone'la
git clone https://github.com/your-org/rmm-platform.git
cd rmm-platform/docker

# 3. Environment ayarla
cp .env.production .env
nano .env   # DB_PASSWORD ve JWT_SECRET değiştir

# 4. Başlat
./deploy.sh
```

## SSL (Let's Encrypt) - Opsiyonel

```bash
# Standalone SSL için Caddy kullan (docker-compose ile)
# Ya da Nginx + certbot:

sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d rmm.yourdomain.com
```

## Servis Durumu

```bash
docker compose ps
docker compose logs -f api
docker compose logs -f frontend
```

## Backup

```bash
docker exec rmm-postgres pg_dump -U rmm rmm_platform > backup_$(date +%Y%m%d).sql
```

## Architecture (CPX32)

```
CPX32 (4 vCPU, 32GB RAM, 320GB NVMe)
├── Nginx (port 80/443) - Frontend static + reverse proxy
├── Go API (port 8080 internal) - Gin + PostgreSQL
└── PostgreSQL 16 - Persistent volume
```

## Firewall (Hetzner Cloud)

- `80/tcp` - HTTP
- `443/tcp` - HTTPS (SSL varsa)
- `22/tcp` - SSH
- Diğer tüm portlar kapalı (PostgreSQL 5432 sadece internal)
