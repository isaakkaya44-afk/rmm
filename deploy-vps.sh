#!/bin/bash
set -e

echo "============================================"
echo " RMM Platform - VPS Deploy (Hetzner CPX32)"
echo "============================================"

# 1. Port 80 çakışan container'ları durdur
echo "[1/14] Stopping conflicting containers..."
docker stop $(docker ps -q --filter "publish=80") 2>/dev/null || true
sleep 2

# 2. Çalışma dizini
echo "[2/14] Preparing directories..."
rm -rf /opt/rmm && mkdir -p /opt/rmm && cd /opt/rmm
mkdir -p backend/cmd/api backend/internal/auth backend/migrations
mkdir -p frontend/src/pages

# 3. docker-compose.yml
echo "[3/14] Creating docker-compose.yml..."
cat > docker-compose.yml << 'EOF'
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: rmm
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: rmm_platform
    volumes: [postgres_data:/var/lib/postgresql/data]
    healthcheck: {test: ["CMD-SHELL", "pg_isready -U rmm"], interval: 5s, timeout: 5s, retries: 5}
    restart: unless-stopped
    networks: [rmm-net]

  api:
    build: {context: ./backend, dockerfile: Dockerfile}
    environment:
      SERVER_PORT: "8080"
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: rmm
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: rmm_platform
      DB_SSLMODE: disable
      JWT_SECRET: ${JWT_SECRET}
    ports: ["127.0.0.1:8080:8080"]
    depends_on: {postgres: {condition: service_healthy}}
    restart: unless-stopped
    networks: [rmm-net]

  frontend:
    build: {context: ./frontend, dockerfile: Dockerfile}
    ports: ["8081:80"]
    depends_on: [api]
    restart: unless-stopped
    networks: [rmm-net]

volumes: {postgres_data: {}}
networks: {rmm-net: {driver: bridge}}
EOF

# 4. Backend Dockerfile
echo "[4/14] Creating backend Dockerfile..."
cat > backend/Dockerfile << 'EOF'
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /rmm-api ./cmd/api
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata postgresql-client
WORKDIR /app
COPY --from=builder /rmm-api .
COPY migrations/ /app/migrations/
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
EOF

# 5. Backend entrypoint.sh
echo "[5/14] Creating entrypoint.sh..."
cat > backend/entrypoint.sh << 'EOF'
#!/bin/sh
set -e
echo "Waiting for postgres..."
for i in $(seq 1 30); do
  pg_isready -h $DB_HOST -U $DB_USER && break || sleep 2
done
echo "Running migrations..."
for f in /app/migrations/*.sql; do
  echo "  Applying: $(basename $f)"
  PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f "$f" 2>/dev/null || true
done
echo "Starting RMM API..."
exec /app/rmm-api
EOF

# 6. go.mod
echo "[6/14] Creating go.mod..."
cat > backend/go.mod << 'EOF'
module rmm-platform
go 1.22
require (
  github.com/gin-gonic/gin v1.10.1
  github.com/golang-jwt/jwt/v5 v5.2.2
  github.com/google/uuid v1.6.0
)
EOF

# 7. main.go
echo "[7/14] Creating main.go..."
cat > backend/cmd/api/main.go << 'EOF'
package main
import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/gin-gonic/gin"
	"rmm-platform/internal/auth"
)
func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"service":"rmm-platform-api","status":"ok","version":"1.0.0"}) })
	r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"message":"RMM Platform API","version":"1.0.0","endpoints":["/health","/api/v1/auth/login","/api/v1/auth/register","/api/v1/dashboard"]}) })
	h := auth.NewHandler()
	api := r.Group("/api/v1")
	{ api.POST("/auth/register", h.Register); api.POST("/auth/login", h.Login); api.GET("/dashboard", h.Dashboard) }
	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatal(err) }
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
}
EOF

# 8. auth.go (DEMO MODE)
echo "[8/14] Creating auth.go (demo mode)..."
cat > backend/internal/auth/auth.go << 'EOF'
package auth
import (
	"os"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)
type Handler struct{}
func NewHandler() *Handler { return &Handler{} }
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Email == "admin@rmm.local" && req.Password == "admin123" {
		claims := Claims{
			UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Email:  req.Email,
			Role:   "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}
		at, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
		rt, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Email:  req.Email,
			Role:   "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
			},
		}).SignedString([]byte(os.Getenv("JWT_SECRET")))
		c.JSON(200, gin.H{"access_token": at, "refresh_token": rt, "token_type": "Bearer"})
		return
	}
	c.JSON(401, gin.H{"error": "invalid credentials"})
}
func (h *Handler) Register(c *gin.Context) { c.JSON(201, gin.H{"message": "registered"}) }
func (h *Handler) Me(c *gin.Context) { c.JSON(200, gin.H{"email": "admin@rmm.local", "role": "admin"}) }
func (h *Handler) Dashboard(c *gin.Context) { c.JSON(200, gin.H{"devices": 0, "alerts": 0, "tickets": 0}) }
func AuthMiddleware() gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
EOF

# 9. Migration
echo "[9/14] Creating migration SQL..."
cat > backend/migrations/001_initial_schema.sql << 'EOF'
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  full_name VARCHAR(255),
  role VARCHAR(50) DEFAULT 'technician',
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS customers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  contact_email VARCHAR(255),
  contact_phone VARCHAR(50),
  address TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
  hostname VARCHAR(255) NOT NULL,
  os VARCHAR(100),
  os_version VARCHAR(100),
  arch VARCHAR(50),
  ip_address INET,
  mac_address MACADDR,
  agent_version VARCHAR(50),
  status VARCHAR(50) DEFAULT 'offline',
  last_heartbeat TIMESTAMPTZ,
  tags TEXT[],
  metadata JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS alerts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
  type VARCHAR(50) NOT NULL,
  severity VARCHAR(20) NOT NULL,
  message TEXT NOT NULL,
  metric_value DOUBLE PRECISION,
  threshold_value DOUBLE PRECISION,
  acknowledged BOOLEAN DEFAULT false,
  acknowledged_by UUID REFERENCES users(id),
  acknowledged_at TIMESTAMPTZ,
  resolved BOOLEAN DEFAULT false,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL,
  device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
  customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  status VARCHAR(50) DEFAULT 'open',
  priority VARCHAR(20) DEFAULT 'medium',
  assigned_to UUID REFERENCES users(id),
  created_by UUID REFERENCES users(id),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  closed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_devices_customer ON devices(customer_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_alerts_device ON alerts(device_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at);
CREATE INDEX IF NOT EXISTS idx_tickets_device ON tickets(device_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
INSERT INTO users (email, password_hash, full_name, role, is_active)
VALUES ('admin@rmm.local', 'demo_no_bcrypt', 'Admin', 'admin', true)
ON CONFLICT (email) DO NOTHING;
EOF

# 10. Frontend Dockerfile
echo "[10/14] Creating frontend files..."
cat > frontend/Dockerfile << 'EOF'
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build
FROM nginx:1.25-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
EOF

# 11. Frontend package.json
cat > frontend/package.json << 'EOF'
{"name":"rmm-frontend","version":"1.0.0","scripts":{"dev":"vite","build":"vite build","preview":"vite preview"},"dependencies":{"react":"^18.2.0","react-dom":"^18.2.0","react-router-dom":"^6.20.0"},"devDependencies":{"@vitejs/plugin-react":"^4.2.0","vite":"^5.0.0"}}
EOF
cat > frontend/package-lock.json << 'EOF'
{"name":"rmm-frontend","version":"1.0.0","lockfileVersion":3,"requires":true,"packages":{}}
EOF
cat > frontend/vite.config.ts << 'EOF'
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
export default defineConfig({ plugins: [react()], build: { outDir: "dist" } })
EOF
cat > frontend/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>RMM Platform</title>
</head>
<body>
  <div id="root"></div>
  <script type="module" src="/src/main.tsx"></script>
</body>
</html>
EOF
cat > frontend/src/main.tsx << 'EOF'
import React from "react"
import ReactDOM from "react-dom/client"
import App from "./App"
ReactDOM.createRoot(document.getElementById("root")!).render(<React.StrictMode><App /></React.StrictMode>)
EOF
cat > frontend/src/App.tsx << 'EOF'
import { BrowserRouter, Routes, Route } from "react-router-dom"
import Login from "./pages/Login"
import Dashboard from "./pages/Dashboard"
export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/*" element={<Dashboard />} />
      </Routes>
    </BrowserRouter>
  )
}
EOF
cat > frontend/src/pages/Login.tsx << 'EOF'
import { useState } from "react"
import { useNavigate } from "react-router-dom"
export default function Login() {
  const [email, setEmail] = useState("admin@rmm.local")
  const [password, setPassword] = useState("admin123")
  const [error, setError] = useState("")
  const navigate = useNavigate()
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password })
      })
      const data = await res.json()
      if (data.access_token) {
        localStorage.setItem("token", data.access_token)
        navigate("/")
      } else {
        setError(data.error || "Login failed")
      }
    } catch {
      setError("Connection error")
    }
  }
  return (
    <div style={{ maxWidth: 400, margin: "100px auto", padding: 20, fontFamily: "sans-serif" }}>
      <h2>RMM Platform Login</h2>
      {error && <div style={{ color: "red", marginBottom: 10 }}>{error}</div>}
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: 15 }}>
          <label style={{ display: "block", marginBottom: 5 }}>Email</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} style={{ width: "100%", padding: 8 }} />
        </div>
        <div style={{ marginBottom: 15 }}>
          <label style={{ display: "block", marginBottom: 5 }}>Password</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} style={{ width: "100%", padding: 8 }} />
        </div>
        <button type="submit" style={{ width: "100%", padding: 10 }}>Login</button>
      </form>
    </div>
  )
}
EOF
cat > frontend/src/pages/Dashboard.tsx << 'EOF'
import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
export default function Dashboard() {
  const [stats, setStats] = useState({ devices: 0, alerts: 0, tickets: 0 })
  const navigate = useNavigate()
  useEffect(() => {
    const token = localStorage.getItem("token")
    if (!token) { navigate("/login"); return }
    fetch("/api/v1/dashboard", { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(setStats)
      .catch(() => navigate("/login"))
  }, [navigate])
  return (
    <div style={{ padding: 20, fontFamily: "sans-serif" }}>
      <h1>RMM Platform Dashboard</h1>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 20, marginTop: 20 }}>
        <div style={{ border: "1px solid #ddd", padding: 20, borderRadius: 8 }}>
          <h3>Devices</h3>
          <p style={{ fontSize: 32, margin: 0 }}>{stats.devices}</p>
        </div>
        <div style={{ border: "1px solid #ddd", padding: 20, borderRadius: 8 }}>
          <h3>Alerts</h3>
          <p style={{ fontSize: 32, margin: 0 }}>{stats.alerts}</p>
        </div>
        <div style={{ border: "1px solid #ddd", padding: 20, borderRadius: 8 }}>
          <h3>Tickets</h3>
          <p style={{ fontSize: 32, margin: 0 }}>{stats.tickets}</p>
        </div>
      </div>
    </div>
  )
}
EOF

# 12. Frontend nginx.conf
echo "[12/14] Creating frontend nginx config..."
cat > frontend/nginx.conf << 'EOF'
server {
  listen 80;
  server_name _;
  root /usr/share/nginx/html;
  index index.html;
  location /api/ {
    proxy_pass http://api:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
  location /ws {
    proxy_pass http://api:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 86400s;
  }
  location / {
    try_files $uri $uri/ /index.html;
  }
}
EOF

# 13. .env
echo "[13/14] Generating secrets..."
DB_PASS=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -base64 32)
cat > .env <<EOF
DB_PASSWORD=$DB_PASS
JWT_SECRET=$JWT_SECRET
EOF
echo "DB_PASSWORD=$DB_PASS"
echo "JWT_SECRET=$JWT_SECRET"

# 14. DEPLOY
echo "[14/14] Building and starting containers..."
export $(grep -v '^#' .env | xargs)
docker compose build --progress=plain 2>&1 | tail -50
docker compose up -d

echo ""
echo "============================================"
echo " DEPLOY COMPLETE"
echo "============================================"
echo "Frontend: http://178.105.87.3:8081"
echo "Login:    admin@rmm.local / admin123"
echo "============================================"
