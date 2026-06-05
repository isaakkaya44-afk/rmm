const express = require('express');
const cors = require('cors');
const http = require('http');
const { WebSocketServer } = require('ws');
const jwt = require('jsonwebtoken');
const { v4: uuidv4 } = require('uuid');
const bcrypt = require('bcryptjs');
const Database = require('better-sqlite3');
const path = require('path');

const DB_PATH = process.env.DB_PATH || path.join(__dirname, '..', 'data', 'rmm.db');
const JWT_SECRET = process.env.JWT_SECRET || 'rmm-platform-secret-key-change-in-production';
const PORT = process.env.SERVER_PORT || 8080;

// Initialize database
const fs = require('fs');
const dbDir = path.dirname(DB_PATH);
if (!fs.existsSync(dbDir)) fs.mkdirSync(dbDir, { recursive: true });

const db = new Database(DB_PATH);
db.pragma('journal_mode = WAL');
db.pragma('foreign_keys = ON');

// Create tables
db.exec(`
  CREATE TABLE IF NOT EXISTS roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    role_id INTEGER NOT NULL REFERENCES roles(id),
    is_active INTEGER DEFAULT 1,
    last_login_at TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS customers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    contact_name TEXT,
    contact_email TEXT,
    contact_phone TEXT,
    address TEXT,
    is_active INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    customer_id TEXT REFERENCES customers(id),
    hostname TEXT NOT NULL,
    os_version TEXT,
    cpu_model TEXT,
    cpu_cores INTEGER,
    ram_total_mb INTEGER,
    disk_total_mb INTEGER,
    rustdesk_id TEXT,
    rustdesk_password TEXT,
    mssql_status TEXT DEFAULT 'unknown',
    pos_process_status TEXT DEFAULT 'unknown',
    agent_version TEXT,
    last_heartbeat TEXT,
    is_online INTEGER DEFAULT 0,
    is_active INTEGER DEFAULT 1,
    tags TEXT,
    notes TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS device_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    cpu_percent REAL,
    ram_percent REAL,
    ram_used_mb INTEGER,
    disk_percent REAL,
    disk_used_mb INTEGER,
    uptime_seconds INTEGER,
    pos_running INTEGER,
    mssql_running INTEGER,
    recorded_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'warning',
    title TEXT NOT NULL,
    message TEXT,
    metric_value REAL,
    threshold_value REAL,
    status TEXT NOT NULL DEFAULT 'open',
    acknowledged_at TEXT,
    acknowledged_by TEXT REFERENCES users(id),
    resolved_at TEXT,
    resolved_by TEXT REFERENCES users(id),
    resolution_note TEXT,
    created_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS tickets (
    id TEXT PRIMARY KEY,
    ticket_number INTEGER UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    priority TEXT NOT NULL DEFAULT 'medium',
    source TEXT DEFAULT 'manual',
    device_id TEXT REFERENCES devices(id),
    customer_id TEXT REFERENCES customers(id),
    assigned_to TEXT REFERENCES users(id),
    created_by TEXT REFERENCES users(id),
    alert_id TEXT REFERENCES alerts(id),
    resolved_at TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS ticket_comments (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    is_internal INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS ticket_attachments (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_size INTEGER,
    content_type TEXT,
    storage_path TEXT,
    uploaded_by TEXT REFERENCES users(id),
    created_at TEXT DEFAULT (datetime('now'))
  );

  CREATE TABLE IF NOT EXISTS remote_sessions (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    technician_id TEXT REFERENCES users(id),
    session_type TEXT NOT NULL DEFAULT 'rustdesk',
    session_id TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    started_at TEXT DEFAULT (datetime('now')),
    ended_at TEXT,
    duration_seconds INTEGER,
    notes TEXT
  );

  CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now'))
  );
`);

// Seed data
const roleCount = db.prepare('SELECT COUNT(*) as c FROM roles').get().c;
if (roleCount === 0) {
  const insertRole = db.prepare('INSERT INTO roles (name, description) VALUES (?, ?)');
  insertRole.run('admin', 'System administrator with full access');
  insertRole.run('technician', 'Support technician with device and ticket access');
  insertRole.run('customer', 'Customer with limited view access');

  const adminHash = bcrypt.hashSync('admin123', 10);
  db.prepare('INSERT INTO users (id, email, password_hash, full_name, role_id) VALUES (?, ?, ?, ?, ?)')
    .run(uuidv4(), 'admin@rmm.local', adminHash, 'System Admin', 1);

  console.log('Seed data created (admin@rmm.local / admin123)');
}

// Ticket number sequence
const ticketSeq = db.prepare('SELECT COALESCE(MAX(ticket_number), 0) + 1 as next FROM tickets');

// App setup
const app = express();
app.use(cors());
app.use(express.json({ limit: '50mb' }));

const server = http.createServer(app);
const wss = new WebSocketServer({ server, path: '/ws' });

// WebSocket broadcast
function broadcast(type, payload) {
  const msg = JSON.stringify({ type, payload });
  wss.clients.forEach(client => {
    if (client.readyState === 1) client.send(msg);
  });
}

// JWT middleware
function authMiddleware(req, res, next) {
  const auth = req.headers.authorization;
  if (!auth || !auth.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Authorization header required' });
  }
  try {
    const decoded = jwt.verify(auth.split(' ')[1], JWT_SECRET);
    req.user = decoded;
    next();
  } catch (err) {
    return res.status(401).json({ error: 'Invalid or expired token' });
  }
}

function roleMiddleware(...roles) {
  return (req, res, next) => {
    if (!roles.includes(req.user.role)) {
      return res.status(403).json({ error: 'Insufficient permissions' });
    }
    next();
  };
}

// ==================== AUTH ROUTES ====================
app.post('/api/v1/auth/login', (req, res) => {
  const { email, password } = req.body;
  const user = db.prepare(`SELECT u.*, r.name as role_name FROM users u JOIN roles r ON r.id = u.role_id WHERE u.email = ? AND u.is_active = 1`).get(email);
  if (!user || !bcrypt.compareSync(password, user.password_hash)) {
    return res.status(401).json({ error: 'Invalid credentials' });
  }

  const token = jwt.sign({ uid: user.id, email: user.email, role: user.role_name }, JWT_SECRET, { expiresIn: '15m' });
  const refreshToken = uuidv4() + '-' + uuidv4();
  const refreshHash = require('crypto').createHash('sha256').update(refreshToken).digest('hex');

  const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
  db.prepare('INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)').run(user.id, refreshHash, expiresAt);
  db.prepare('UPDATE users SET last_login_at = datetime(\'now\') WHERE id = ?').run(user.id);

  res.json({
    access_token: token,
    refresh_token: refreshToken,
    token_type: 'Bearer',
    expires_in: 900,
    user: { id: user.id, email: user.email, full_name: user.full_name, role: user.role_name, is_active: !!user.is_active }
  });
});

app.post('/api/v1/auth/refresh', (req, res) => {
  const { refresh_token } = req.body;
  const hash = require('crypto').createHash('sha256').update(refresh_token).digest('hex');
  const rt = db.prepare('SELECT * FROM refresh_tokens WHERE token_hash = ? AND expires_at > datetime(\'now\')').get(hash);
  if (!rt) return res.status(401).json({ error: 'Invalid or expired refresh token' });

  const user = db.prepare(`SELECT u.*, r.name as role_name FROM users u JOIN roles r ON r.id = u.role_id WHERE u.id = ?`).get(rt.user_id);
  if (!user) return res.status(401).json({ error: 'User not found' });

  db.prepare('DELETE FROM refresh_tokens WHERE id = ?').run(rt.id);

  const token = jwt.sign({ uid: user.id, email: user.email, role: user.role_name }, JWT_SECRET, { expiresIn: '15m' });
  const newRefresh = uuidv4() + '-' + uuidv4();
  const newHash = require('crypto').createHash('sha256').update(newRefresh).digest('hex');
  const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
  db.prepare('INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)').run(user.id, newHash, expiresAt);

  res.json({
    access_token: token,
    refresh_token: newRefresh,
    token_type: 'Bearer',
    expires_in: 900,
    user: { id: user.id, email: user.email, full_name: user.full_name, role: user.role_name, is_active: !!user.is_active }
  });
});

app.post('/api/v1/auth/logout', (req, res) => {
  const { refresh_token } = req.body;
  if (refresh_token) {
    const hash = require('crypto').createHash('sha256').update(refresh_token).digest('hex');
    db.prepare('DELETE FROM refresh_tokens WHERE token_hash = ?').run(hash);
  }
  res.json({ message: 'logged out successfully' });
});

app.get('/api/v1/me', authMiddleware, (req, res) => {
  const user = db.prepare(`SELECT u.*, r.name as role_name FROM users u JOIN roles r ON r.id = u.role_id WHERE u.id = ?`).get(req.user.uid);
  if (!user) return res.status(404).json({ error: 'User not found' });
  res.json({ id: user.id, email: user.email, full_name: user.full_name, role: user.role_name, is_active: !!user.is_active });
});

// ==================== DEVICE ROUTES ====================
app.get('/api/v1/devices', authMiddleware, (req, res) => {
  const page = parseInt(req.query.page) || 1;
  const limit = Math.min(parseInt(req.query.limit) || 50, 100);
  const search = req.query.search || '';
  const offset = (page - 1) * limit;

  let where = 'WHERE d.is_active = 1';
  let params = [];
  if (search) { where += ' AND d.hostname LIKE ?'; params.push(`%${search}%`); }

  const total = db.prepare(`SELECT COUNT(*) as c FROM devices d ${where}`).get(...params).c;
  const devices = db.prepare(`SELECT d.*, c.name as customer_name FROM devices d LEFT JOIN customers c ON c.id = d.customer_id ${where} ORDER BY d.last_heartbeat DESC NULLS LAST LIMIT ? OFFSET ?`).all(...params, limit, offset);

  res.json({ devices, total, page, limit });
});

app.get('/api/v1/devices/:id', authMiddleware, (req, res) => {
  const device = db.prepare(`SELECT d.*, c.name as customer_name FROM devices d LEFT JOIN customers c ON c.id = d.customer_id WHERE d.id = ?`).get(req.params.id);
  if (!device) return res.status(404).json({ error: 'Device not found' });
  res.json(device);
});

app.post('/api/v1/devices', authMiddleware, (req, res) => {
  const id = uuidv4();
  const { hostname, customer_id, os_version, rustdesk_id, notes } = req.body;
  if (!hostname) return res.status(400).json({ error: 'hostname is required' });
  db.prepare('INSERT INTO devices (id, hostname, customer_id, os_version, rustdesk_id, notes) VALUES (?, ?, ?, ?, ?, ?)').run(id, hostname, customer_id||null, os_version||null, rustdesk_id||null, notes||null);
  const device = db.prepare('SELECT * FROM devices WHERE id = ?').get(id);
  broadcast('device.created', device);
  res.status(201).json(device);
});

app.put('/api/v1/devices/:id', authMiddleware, (req, res) => {
  const { hostname, customer_id, rustdesk_id, rustdesk_password, mssql_status, pos_process_status, notes, tags } = req.body;
  const fields = []; const params = [];
  if (hostname !== undefined) { fields.push('hostname = ?'); params.push(hostname); }
  if (customer_id !== undefined) { fields.push('customer_id = ?'); params.push(customer_id); }
  if (rustdesk_id !== undefined) { fields.push('rustdesk_id = ?'); params.push(rustdesk_id); }
  if (rustdesk_password !== undefined) { fields.push('rustdesk_password = ?'); params.push(rustdesk_password); }
  if (mssql_status !== undefined) { fields.push('mssql_status = ?'); params.push(mssql_status); }
  if (pos_process_status !== undefined) { fields.push('pos_process_status = ?'); params.push(pos_process_status); }
  if (notes !== undefined) { fields.push('notes = ?'); params.push(notes); }
  if (tags !== undefined) { fields.push('tags = ?'); params.push(JSON.stringify(tags)); }
  fields.push("updated_at = datetime('now')");
  params.push(req.params.id);
  db.prepare(`UPDATE devices SET ${fields.join(', ')} WHERE id = ?`).run(...params);
  const device = db.prepare('SELECT * FROM devices WHERE id = ?').get(req.params.id);
  broadcast('device.updated', device);
  res.json(device);
});

app.delete('/api/v1/devices/:id', authMiddleware, (req, res) => {
  db.prepare("UPDATE devices SET is_active = 0, updated_at = datetime('now') WHERE id = ?").run(req.params.id);
  broadcast('device.deleted', { id: req.params.id });
  res.json({ message: 'device deleted' });
});

app.get('/api/v1/devices/:id/metrics', authMiddleware, (req, res) => {
  const limit = parseInt(req.query.limit) || 60;
  const metrics = db.prepare('SELECT * FROM device_metrics WHERE device_id = ? ORDER BY recorded_at DESC LIMIT ?').all(req.params.id, limit);
  res.json(metrics);
});

// Public heartbeat endpoint (no auth required)
app.post('/api/v1/devices/heartbeat', (req, res) => {
  const { hostname, os_version, cpu_model, cpu_cores, ram_total_mb, ram_used_mb, ram_percent, disk_total_mb, disk_used_mb, disk_percent, cpu_percent, uptime_seconds, rustdesk_id, pos_running, mssql_running, agent_version } = req.body;
  if (!hostname) return res.status(400).json({ error: 'hostname required' });

  let device = db.prepare('SELECT * FROM devices WHERE hostname = ?').get(hostname);
  if (!device) {
    const id = uuidv4();
    db.prepare('INSERT INTO devices (id, hostname, os_version, rustdesk_id, agent_version, is_online, last_heartbeat) VALUES (?, ?, ?, ?, ?, 1, datetime(\'now\'))').run(id, hostname, os_version||null, rustdesk_id||null, agent_version||null);
    device = db.prepare('SELECT * FROM devices WHERE id = ?').get(id);
  } else {
    db.prepare(`UPDATE devices SET os_version = COALESCE(?, os_version), cpu_model = COALESCE(?, cpu_model), cpu_cores = COALESCE(?, cpu_cores), ram_total_mb = COALESCE(?, ram_total_mb), disk_total_mb = COALESCE(?, disk_total_mb), rustdesk_id = COALESCE(?, rustdesk_id), agent_version = COALESCE(?, agent_version), is_online = 1, last_heartbeat = datetime('now'), updated_at = datetime('now') WHERE id = ?`)
      .run(os_version||null, cpu_model||null, cpu_cores||null, ram_total_mb||null, disk_total_mb||null, rustdesk_id||null, agent_version||null, device.id);

    db.prepare('INSERT INTO device_metrics (device_id, cpu_percent, ram_percent, ram_used_mb, disk_percent, disk_used_mb, uptime_seconds, pos_running, mssql_running) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)')
      .run(device.id, cpu_percent||null, ram_percent||null, ram_used_mb||null, disk_percent||null, disk_used_mb||null, uptime_seconds||null, pos_running ? 1 : 0, mssql_running ? 1 : 0);

    // Check alert thresholds
    checkAlerts(device.id, cpu_percent, ram_percent, disk_percent, pos_running, mssql_running);
  }

  broadcast('device.heartbeat', device);
  res.json(device);
});

// Alert engine
function checkAlerts(deviceId, cpu, ram, disk, posRunning, mssqlRunning) {
  const alerts = [];

  function hasActive(type) {
    return !!db.prepare("SELECT id FROM alerts WHERE device_id = ? AND type = ? AND status IN ('open','acknowledged')").get(deviceId, type);
  }

  if (cpu && cpu > 90 && !hasActive('cpu_high')) {
    const id = uuidv4();
    db.prepare("INSERT INTO alerts (id, device_id, type, severity, title, message, metric_value, threshold_value) VALUES (?, ?, 'cpu_high', 'critical', 'CPU usage exceeds 90% threshold', ?, ?, 90)").run(id, deviceId, `CPU usage: ${cpu.toFixed(1)}%`, cpu);
    alerts.push(db.prepare('SELECT * FROM alerts WHERE id = ?').get(id));
  }
  if (ram && ram > 90 && !hasActive('ram_high')) {
    const id = uuidv4();
    db.prepare("INSERT INTO alerts (id, device_id, type, severity, title, message, metric_value, threshold_value) VALUES (?, ?, 'ram_high', 'warning', 'RAM usage exceeds 90% threshold', ?, ?, 90)").run(id, deviceId, `RAM usage: ${ram.toFixed(1)}%`, ram);
    alerts.push(db.prepare('SELECT * FROM alerts WHERE id = ?').get(id));
  }
  if (disk && disk > 90 && !hasActive('disk_high')) {
    const id = uuidv4();
    db.prepare("INSERT INTO alerts (id, device_id, type, severity, title, message, metric_value, threshold_value) VALUES (?, ?, 'disk_high', 'warning', 'Disk usage exceeds 90% threshold', ?, ?, 90)").run(id, deviceId, `Disk usage: ${disk.toFixed(1)}%`, disk);
    alerts.push(db.prepare('SELECT * FROM alerts WHERE id = ?').get(id));
  }
  if (!posRunning && !hasActive('pos_process_down')) {
    const id = uuidv4();
    db.prepare("INSERT INTO alerts (id, device_id, type, severity, title, status) VALUES (?, ?, 'pos_process_down', 'critical', 'POS process is not running', 'open')").run(id, deviceId);
    alerts.push(db.prepare('SELECT * FROM alerts WHERE id = ?').get(id));
  }
  if (!mssqlRunning && !hasActive('mssql_stopped')) {
    const id = uuidv4();
    db.prepare("INSERT INTO alerts (id, device_id, type, severity, title, status) VALUES (?, ?, 'mssql_stopped', 'critical', 'MSSQL service is stopped', 'open')").run(id, deviceId);
    alerts.push(db.prepare('SELECT * FROM alerts WHERE id = ?').get(id));
  }

  alerts.forEach(a => broadcast('alert.created', a));
}

// Offline detection (run every 30s)
setInterval(() => {
  const devices = db.prepare("SELECT id, hostname FROM devices WHERE is_online = 1 AND datetime('now') > datetime(last_heartbeat, '+2 minutes')").all();
  devices.forEach(d => {
    db.prepare("UPDATE devices SET is_online = 0, updated_at = datetime('now') WHERE id = ?").run(d.id);
    broadcast('device.offline', d);
    if (!db.prepare("SELECT id FROM alerts WHERE device_id = ? AND type = 'device_offline' AND status IN ('open','acknowledged')").get(d.id)) {
      const id = uuidv4();
      db.prepare("INSERT INTO alerts (id, device_id, type, severity, title) VALUES (?, ?, 'device_offline', 'critical', 'Device is offline')").run(id, d.id);
      broadcast('alert.created', db.prepare('SELECT * FROM alerts WHERE id = ?').get(id));
    }
  });
}, 30000);

// ==================== ALERT ROUTES ====================
app.get('/api/v1/alerts', authMiddleware, (req, res) => {
  const page = parseInt(req.query.page) || 1;
  const limit = Math.min(parseInt(req.query.limit) || 50, 100);
  const { status, severity, device_id } = req.query;
  let where = 'WHERE 1=1'; const params = [];
  if (status) { where += ' AND a.status = ?'; params.push(status); }
  if (severity) { where += ' AND a.severity = ?'; params.push(severity); }
  if (device_id) { where += ' AND a.device_id = ?'; params.push(device_id); }
  const offset = (page - 1) * limit;
  const total = db.prepare(`SELECT COUNT(*) as c FROM alerts a ${where}`).get(...params).c;
  const alerts = db.prepare(`SELECT a.*, d.hostname as device_name FROM alerts a LEFT JOIN devices d ON d.id = a.device_id ${where} ORDER BY a.created_at DESC LIMIT ? OFFSET ?`).all(...params, limit, offset);
  res.json({ alerts, total });
});

app.get('/api/v1/alerts/:id', authMiddleware, (req, res) => {
  const alert = db.prepare('SELECT a.*, d.hostname as device_name FROM alerts a LEFT JOIN devices d ON d.id = a.device_id WHERE a.id = ?').get(req.params.id);
  if (!alert) return res.status(404).json({ error: 'Alert not found' });
  res.json(alert);
});

app.post('/api/v1/alerts/:id/acknowledge', authMiddleware, (req, res) => {
  db.prepare("UPDATE alerts SET status = 'acknowledged', acknowledged_at = datetime('now'), acknowledged_by = ? WHERE id = ?").run(req.user.uid, req.params.id);
  const alert = db.prepare('SELECT * FROM alerts WHERE id = ?').get(req.params.id);
  broadcast('alert.updated', alert);
  res.json(alert);
});

app.post('/api/v1/alerts/:id/resolve', authMiddleware, (req, res) => {
  const { resolution_note } = req.body || {};
  db.prepare("UPDATE alerts SET status = 'resolved', resolved_at = datetime('now'), resolved_by = ?, resolution_note = ? WHERE id = ?").run(req.user.uid, resolution_note||null, req.params.id);
  const alert = db.prepare('SELECT * FROM alerts WHERE id = ?').get(req.params.id);
  broadcast('alert.updated', alert);
  res.json(alert);
});

// ==================== TICKET ROUTES ====================
app.get('/api/v1/tickets', authMiddleware, (req, res) => {
  const page = parseInt(req.query.page) || 1;
  const limit = Math.min(parseInt(req.query.limit) || 50, 100);
  const { status, device_id } = req.query;
  let where = 'WHERE 1=1'; const params = [];
  if (status) { where += ' AND t.status = ?'; params.push(status); }
  if (device_id) { where += ' AND t.device_id = ?'; params.push(device_id); }
  const offset = (page - 1) * limit;
  const total = db.prepare(`SELECT COUNT(*) as c FROM tickets t ${where}`).get(...params).c;
  const tickets = db.prepare(`SELECT t.*, d.hostname as device_name, c.name as customer_name, u1.full_name as assigned_name, u2.full_name as created_name FROM tickets t LEFT JOIN devices d ON d.id = t.device_id LEFT JOIN customers c ON c.id = t.customer_id LEFT JOIN users u1 ON u1.id = t.assigned_to LEFT JOIN users u2 ON u2.id = t.created_by ${where} ORDER BY t.created_at DESC LIMIT ? OFFSET ?`).all(...params, limit, offset);
  res.json({ tickets, total, page, limit });
});

app.get('/api/v1/tickets/:id', authMiddleware, (req, res) => {
  const ticket = db.prepare(`SELECT t.*, d.hostname as device_name, c.name as customer_name, u1.full_name as assigned_name, u2.full_name as created_name FROM tickets t LEFT JOIN devices d ON d.id = t.device_id LEFT JOIN customers c ON c.id = t.customer_id LEFT JOIN users u1 ON u1.id = t.assigned_to LEFT JOIN users u2 ON u2.id = t.created_by WHERE t.id = ?`).get(req.params.id);
  if (!ticket) return res.status(404).json({ error: 'Ticket not found' });
  const comments = db.prepare('SELECT tc.*, u.full_name as user_name FROM ticket_comments tc LEFT JOIN users u ON u.id = tc.user_id WHERE tc.ticket_id = ? ORDER BY tc.created_at ASC').all(req.params.id);
  res.json({ ticket, comments });
});

app.post('/api/v1/tickets', authMiddleware, (req, res) => {
  const { title, description, priority, source, device_id, customer_id, assigned_to, alert_id } = req.body;
  if (!title) return res.status(400).json({ error: 'title is required' });
  const id = uuidv4();
  const nextNum = ticketSeq.get().next;
  db.prepare('INSERT INTO tickets (id, ticket_number, title, description, priority, source, device_id, customer_id, assigned_to, created_by, alert_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)')
    .run(id, nextNum, title, description||null, priority||'medium', source||'manual', device_id||null, customer_id||null, assigned_to||null, req.user.uid, alert_id||null);
  const ticket = db.prepare('SELECT * FROM tickets WHERE id = ?').get(id);
  broadcast('ticket.created', ticket);
  res.status(201).json(ticket);
});

app.put('/api/v1/tickets/:id', authMiddleware, (req, res) => {
  const { title, description, status, priority, assigned_to } = req.body;
  const fields = ["updated_at = datetime('now')"]; const params = [];
  if (title !== undefined) { fields.push('title = ?'); params.push(title); }
  if (description !== undefined) { fields.push('description = ?'); params.push(description); }
  if (status !== undefined) { fields.push('status = ?'); params.push(status); if (status === 'resolved') fields.push("resolved_at = datetime('now')"); }
  if (priority !== undefined) { fields.push('priority = ?'); params.push(priority); }
  if (assigned_to !== undefined) { fields.push('assigned_to = ?'); params.push(assigned_to); }
  params.push(req.params.id);
  db.prepare(`UPDATE tickets SET ${fields.join(', ')} WHERE id = ?`).run(...params);
  const ticket = db.prepare('SELECT * FROM tickets WHERE id = ?').get(req.params.id);
  broadcast('ticket.updated', ticket);
  res.json(ticket);
});

app.post('/api/v1/tickets/:id/comments', authMiddleware, (req, res) => {
  const { content, is_internal } = req.body;
  if (!content) return res.status(400).json({ error: 'content is required' });
  const id = uuidv4();
  db.prepare('INSERT INTO ticket_comments (id, ticket_id, user_id, content, is_internal) VALUES (?, ?, ?, ?, ?)').run(id, req.params.id, req.user.uid, content, is_internal ? 1 : 0);
  const comment = db.prepare('SELECT tc.*, u.full_name as user_name FROM ticket_comments tc LEFT JOIN users u ON u.id = tc.user_id WHERE tc.id = ?').get(id);
  broadcast('ticket.comment_added', comment);
  res.status(201).json(comment);
});

// ==================== CUSTOMER ROUTES ====================
app.get('/api/v1/customers', authMiddleware, (req, res) => {
  const customers = db.prepare('SELECT * FROM customers WHERE is_active = 1 ORDER BY name').all();
  res.json(customers);
});

app.get('/api/v1/customers/:id', authMiddleware, (req, res) => {
  const customer = db.prepare('SELECT * FROM customers WHERE id = ?').get(req.params.id);
  if (!customer) return res.status(404).json({ error: 'Customer not found' });
  res.json(customer);
});

app.post('/api/v1/customers', authMiddleware, (req, res) => {
  const { name, contact_name, contact_email, contact_phone, address } = req.body;
  if (!name) return res.status(400).json({ error: 'name is required' });
  const id = uuidv4();
  db.prepare('INSERT INTO customers (id, name, contact_name, contact_email, contact_phone, address) VALUES (?, ?, ?, ?, ?, ?)').run(id, name, contact_name||null, contact_email||null, contact_phone||null, address||null);
  res.status(201).json(db.prepare('SELECT * FROM customers WHERE id = ?').get(id));
});

// ==================== REMOTE SESSION ROUTES ====================
app.get('/api/v1/remote/sessions/:deviceId', authMiddleware, (req, res) => {
  const sessions = db.prepare('SELECT rs.*, u.full_name as technician_name FROM remote_sessions rs LEFT JOIN users u ON u.id = rs.technician_id WHERE rs.device_id = ? ORDER BY rs.started_at DESC').all(req.params.deviceId);
  res.json(sessions);
});

app.post('/api/v1/remote/sessions', authMiddleware, (req, res) => {
  const { device_id, session_type, session_id } = req.body;
  if (!device_id) return res.status(400).json({ error: 'device_id is required' });
  const id = uuidv4();
  db.prepare("INSERT INTO remote_sessions (id, device_id, technician_id, session_type, session_id) VALUES (?, ?, ?, ?, ?)").run(id, device_id, req.user.uid, session_type||'rustdesk', session_id||null);
  res.status(201).json(db.prepare('SELECT * FROM remote_sessions WHERE id = ?').get(id));
});

// ==================== DASHBOARD ====================
app.get('/api/v1/dashboard', authMiddleware, (req, res) => {
  const total = db.prepare("SELECT COUNT(*) as c FROM devices WHERE is_active = 1").get().c;
  const online = db.prepare("SELECT COUNT(*) as c FROM devices WHERE is_active = 1 AND is_online = 1").get().c;
  const offline = db.prepare("SELECT COUNT(*) as c FROM devices WHERE is_active = 1 AND is_online = 0").get().c;
  const criticalAlerts = db.prepare("SELECT COUNT(*) as c FROM alerts WHERE status IN ('open','acknowledged') AND severity IN ('critical','warning')").get().c;
  const openTickets = db.prepare("SELECT COUNT(*) as c FROM tickets WHERE status NOT IN ('resolved','closed')").get().c;
  res.json({ total_devices: total, online_devices: online, offline_devices: offline, critical_alerts: criticalAlerts, open_tickets: openTickets });
});

// ==================== HEALTH ====================
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'rmm-platform-api', version: '1.0.0' });
});

// Start server
server.listen(PORT, () => {
  console.log(`RMM API Server running on http://localhost:${PORT}`);
  console.log(`WebSocket available at ws://localhost:${PORT}/ws`);
  console.log(`Default login: admin@rmm.local / admin123`);
});
