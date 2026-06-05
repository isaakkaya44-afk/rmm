-- RMM Platform Database Schema
-- PostgreSQL Migration 001

BEGIN;

-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin', 'System administrator with full access'),
    ('technician', 'Support technician with device and ticket access'),
    ('customer', 'Customer with limited view access')
ON CONFLICT (name) DO NOTHING;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    role_id         INTEGER NOT NULL REFERENCES roles(id),
    is_active       BOOLEAN DEFAULT TRUE,
    last_login_at   TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role_id);

-- Customers
CREATE TABLE IF NOT EXISTS customers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    contact_name    VARCHAR(255),
    contact_email   VARCHAR(255),
    contact_phone   VARCHAR(50),
    address         TEXT,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Devices
CREATE TABLE IF NOT EXISTS devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID REFERENCES customers(id),
    hostname        VARCHAR(255) NOT NULL,
    os_version      VARCHAR(255),
    cpu_model       VARCHAR(255),
    cpu_cores       INTEGER,
    ram_total_mb    BIGINT,
    disk_total_mb   BIGINT,
    rustdesk_id     VARCHAR(100),
    rustdesk_password VARCHAR(100),
    mssql_status    VARCHAR(50) DEFAULT 'unknown',
    pos_process_status VARCHAR(50) DEFAULT 'unknown',
    agent_version   VARCHAR(50),
    last_heartbeat  TIMESTAMP WITH TIME ZONE,
    is_online       BOOLEAN DEFAULT FALSE,
    is_active       BOOLEAN DEFAULT TRUE,
    tags            TEXT[],
    notes           TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_devices_customer ON devices(customer_id);
CREATE INDEX idx_devices_hostname ON devices(hostname);
CREATE INDEX idx_devices_rustdesk ON devices(rustdesk_id);
CREATE INDEX idx_devices_online ON devices(is_online) WHERE is_online = TRUE;

-- Device Metrics (time-series)
CREATE TABLE IF NOT EXISTS device_metrics (
    id              BIGSERIAL PRIMARY KEY,
    device_id       UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    cpu_percent     DOUBLE PRECISION,
    ram_percent     DOUBLE PRECISION,
    ram_used_mb     BIGINT,
    disk_percent    DOUBLE PRECISION,
    disk_used_mb    BIGINT,
    uptime_seconds  BIGINT,
    pos_running     BOOLEAN,
    mssql_running   BOOLEAN,
    recorded_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_device_metrics_device ON device_metrics(device_id);
CREATE INDEX idx_device_metrics_time ON device_metrics(recorded_at DESC);
CREATE INDEX idx_device_metrics_lookup ON device_metrics(device_id, recorded_at DESC);

-- Alerts
CREATE TABLE IF NOT EXISTS alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    type            VARCHAR(50) NOT NULL,
    severity        VARCHAR(20) NOT NULL DEFAULT 'warning',
    title           VARCHAR(255) NOT NULL,
    message         TEXT,
    metric_value    DOUBLE PRECISION,
    threshold_value DOUBLE PRECISION,
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by UUID REFERENCES users(id),
    resolved_at     TIMESTAMP WITH TIME ZONE,
    resolved_by     UUID REFERENCES users(id),
    resolution_note TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alerts_device ON alerts(device_id);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_created ON alerts(created_at DESC);

-- Tickets
CREATE TABLE IF NOT EXISTS tickets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number   SERIAL UNIQUE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    priority        VARCHAR(20) NOT NULL DEFAULT 'medium',
    source          VARCHAR(50) DEFAULT 'manual',
    device_id       UUID REFERENCES devices(id),
    customer_id     UUID REFERENCES customers(id),
    assigned_to     UUID REFERENCES users(id),
    created_by      UUID REFERENCES users(id),
    alert_id        UUID REFERENCES alerts(id),
    resolved_at     TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_assigned ON tickets(assigned_to);
CREATE INDEX idx_tickets_device ON tickets(device_id);
CREATE INDEX idx_tickets_created ON tickets(created_at DESC);

-- Ticket Comments
CREATE TABLE IF NOT EXISTS ticket_comments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    content         TEXT NOT NULL,
    is_internal     BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_ticket_comments_ticket ON ticket_comments(ticket_id);

-- Ticket Attachments
CREATE TABLE IF NOT EXISTS ticket_attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    file_name       VARCHAR(255) NOT NULL,
    file_size       BIGINT,
    content_type    VARCHAR(100),
    storage_path    TEXT,
    uploaded_by     UUID REFERENCES users(id),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Remote Sessions
CREATE TABLE IF NOT EXISTS remote_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    technician_id   UUID REFERENCES users(id),
    session_type    VARCHAR(50) NOT NULL DEFAULT 'rustdesk',
    session_id      VARCHAR(255),
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    started_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at        TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    notes           TEXT
);

CREATE INDEX idx_remote_sessions_device ON remote_sessions(device_id);
CREATE INDEX idx_remote_sessions_technician ON remote_sessions(technician_id);

-- Refresh Tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id              SERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(255) NOT NULL,
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- Seed default admin user (password: admin123)
INSERT INTO users (email, password_hash, full_name, role_id)
SELECT 'admin@rmm.local', '$2a$10$qNEKxk7WpsW97Yj5vWq3se5cKXrTFGhjS7prwX6Xha64c6m7VLYFW', 'System Admin', r.id
FROM roles r
WHERE r.name = 'admin'
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'admin@rmm.local');

COMMIT;
