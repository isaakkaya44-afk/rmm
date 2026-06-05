# RMM Platform — POS Support & Remote Monitoring

Production-ready MVP for POS service companies managing restaurant POS systems, Windows terminals, MSSQL servers, printers, and remote support operations.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌────────────────┐
│  Windows Agent  │────▶│  Go Backend API  │────▶│  PostgreSQL    │
│  (Go Service)   │     │  (Gin + WebSocket)│    │  (TimescaleDB) │
└─────────────────┘     └──────────────────┘     └────────────────┘
                               │                        │
                               ▼                        ▼
                        ┌──────────────────┐     ┌────────────────┐
                        │  Redis (Cache)   │     │  Grafana       │
                        └──────────────────┘     │  + Prometheus  │
                                                  └────────────────┘
┌─────────────────┐
│  Desktop Panel  │────▶ WebSocket / REST
│  (Tauri + React)│
└─────────────────┘
```

## Tech Stack

| Component   | Technology |
|-------------|-----------|
| Backend     | Go 1.22, Gin, WebSocket (gorilla/websocket) |
| Database    | PostgreSQL 16 |
| Cache       | Redis 7 |
| Agent       | Go 1.22, gopsutil, Windows Service |
| Desktop     | Tauri 1.x, React 18, TypeScript |
| Monitoring  | Prometheus, Grafana (optional) |
| Remote      | RustDesk (external) |
| Auth        | JWT (golang-jwt) |
| Logging     | zerolog |

## Quick Start

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- Node.js 18+ (for desktop)
- Rust toolchain (for Tauri, optional)

### 1. Database & Cache
```bash
docker compose -f docker/docker-compose.yml up -d
```

### 2. Run Migrations
```bash
psql -h localhost -U rmm -d rmm_platform -f backend/migrations/001_initial_schema.sql
```

### 3. Start Backend
```bash
cd backend
cp .env.example .env
go mod download
go run ./cmd/api
```

### 4. Start Desktop
```bash
cd desktop
npm install
npm run dev
```

### 5. Install & Run Agent (Windows)
```powershell
# As Administrator
agent.exe install
agent.exe run
```

## API Endpoints

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/refresh` | Refresh token |
| POST | `/api/v1/auth/logout` | Logout |
| GET | `/api/v1/me` | Current user |

### Devices
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/devices` | List devices |
| GET | `/api/v1/devices/:id` | Device detail |
| POST | `/api/v1/devices` | Register device |
| PUT | `/api/v1/devices/:id` | Update device |
| DELETE | `/api/v1/devices/:id` | Delete device |
| POST | `/api/v1/devices/heartbeat` | Agent heartbeat |
| GET | `/api/v1/devices/:id/metrics` | Device metrics |

### Alerts
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/alerts` | List alerts |
| GET | `/api/v1/alerts/:id` | Alert detail |
| POST | `/api/v1/alerts/:id/acknowledge` | Acknowledge |
| POST | `/api/v1/alerts/:id/resolve` | Resolve |

### Tickets
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tickets` | List tickets |
| GET | `/api/v1/tickets/:id` | Ticket detail |
| POST | `/api/v1/tickets` | Create ticket |
| PUT | `/api/v1/tickets/:id` | Update ticket |
| POST | `/api/v1/tickets/:id/comments` | Add comment |

### Real-time
| Protocol | Path | Description |
|----------|------|-------------|
| WebSocket | `/ws` | Real-time events |

### System
| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/dashboard` | Dashboard summary |

## Agent Configuration

Edit `C:\ProgramData\RMMAgent\config.yaml`:

```yaml
server:
  base_url: "http://your-server:8080"
  api_key: ""

agent:
  interval: 30
  rustdesk_path: "C:\\Program Files\\RustDesk"

monitor:
  pos_processes:
    - "pos.exe"
    - "samba_pos.exe"
  mssql_services:
    - "MSSQLSERVER"
```

## Default Credentials

- Email: `admin@rmm.local`
- Password: `admin123`

## Project Structure

```
rmm-platform/
├── backend/           # Go API backend
│   ├── cmd/api/       # Entry point
│   ├── internal/      # Application packages
│   │   ├── auth/      # JWT auth, RBAC
│   │   ├── devices/   # Device management
│   │   ├── monitoring/ # Health scores
│   │   ├── alerts/    # Alert engine
│   │   ├── tickets/   # Ticket system
│   │   ├── customers/ # Customer management
│   │   ├── remote/    # Remote sessions
│   │   ├── realtime/  # WebSocket hub
│   │   └── shared/    # Config, DB, logging
│   ├── migrations/    # SQL schema
│   └── docker/        # Dockerfile
├── agent/             # Windows agent
│   ├── cmd/           # Entry point
│   ├── core/          # Core modules
│   │   ├── collector/  # System metrics
│   │   ├── monitor/    # POS monitoring
│   │   ├── network/    # Heartbeat client
│   │   ├── remote/     # RustDesk detection
│   │   └── config/     # Configuration
│   ├── service/       # Windows service
│   ├── transport/     # HTTP transport
│   └── utils/         # Screenshot, etc.
├── desktop/           # Tauri + React
│   ├── src/           # React app
│   │   ├── pages/     # Page components
│   │   └── components/ # Shared components
│   └── src-tauri/     # Tauri config
├── docker/            # Docker Compose
├── shared/            # Shared constants
└── docs/              # Documentation
```

## Default User Roles

| Role | Permissions |
|------|-------------|
| admin | Full system access |
| technician | Device & ticket access |
| customer | Limited view access |

## Remote Access

Integrates with self-hosted RustDesk:
1. Agent detects and reports RustDesk ID
2. Desktop app shows "Connect" button
3. Clicking opens `rustdesk://<id>` URI

## Alert Thresholds

| Condition | Threshold |
|-----------|-----------|
| CPU Usage | > 90% for 30s |
| RAM Usage | > 90% for 30s |
| Disk Usage | > 90% for 30s |
| Device Offline | No heartbeat > 2min |
| POS Process | Not running |
| MSSQL Service | Stopped |

## License

MIT
