import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { useAuth, API_BASE } from "../App";

interface Device {
  id: string; hostname: string; os_version: string; is_online: boolean;
  cpu_percent: number; ram_percent: number; disk_percent: number;
  last_heartbeat: string; pos_process_status: string; mssql_status: string;
  rustdesk_id: string; agent_version: string;
}

interface DashboardSummary {
  total_devices: number; online_devices: number; offline_devices: number;
  critical_alerts: number; open_tickets: number;
}

export default function Dashboard({ wsEvents }: { wsEvents: any[] }) {
  const { token } = useAuth();
  const [devices, setDevices] = useState<Device[]>([]);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  const loadData = () => {
    setLoading(true);
    fetch(API_BASE + "/dashboard", { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(setSummary).catch(() => {});

    fetch(API_BASE + "/devices?limit=100", { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => setDevices(d.devices || [])).catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => { loadData(); }, [token]);
  useEffect(() => {
    if (wsEvents.some(e => e.type?.startsWith("device.") || e.type?.startsWith("alert."))) {
      loadData();
    }
  }, [wsEvents]);

  const filtered = devices.filter(d =>
    d.hostname.toLowerCase().includes(search.toLowerCase())
  );

  const cards = [
    { label: "Total Devices", value: summary?.total_devices || 0, color: "#1890ff" },
    { label: "Online", value: summary?.online_devices || 0, color: "#52c41a" },
    { label: "Offline", value: summary?.offline_devices || 0, color: "#ff4d4f" },
    { label: "Critical Alerts", value: summary?.critical_alerts || 0, color: "#fa8c16" },
    { label: "Open Tickets", value: summary?.open_tickets || 0, color: "#722ed1" },
  ];

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontSize: 24 }}>Dashboard</h1>
        <button onClick={loadData} style={{
          padding: "8px 16px", background: "#1890ff", color: "#fff",
          border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13
        }}>⟳ Yenile</button>
      </div>

      {loading && <div style={{ textAlign: "center", padding: 40, color: "#999" }}>Yükleniyor...</div>}

      {summary && (
        <div style={{ display: "flex", gap: 16, marginBottom: 24 }}>
          {cards.map(card => (
            <div key={card.label} style={{
              flex: 1, background: "#fff", padding: 20, borderRadius: 8,
              borderTop: `3px solid ${card.color}`,
              boxShadow: "0 1px 3px rgba(0,0,0,0.08)",
            }}>
              <h3 style={{ fontSize: 28, margin: 0, color: card.color }}>{card.value}</h3>
              <p style={{ margin: "4px 0 0", fontSize: 13, color: "#666" }}>{card.label}</p>
            </div>
          ))}
        </div>
      )}

      <div style={{
        display: "flex", justifyContent: "space-between", alignItems: "center",
        marginBottom: 16, gap: 12,
      }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>Cihazlar</h2>
        <div style={{ display: "flex", gap: 8 }}>
          <select style={{
            padding: "8px 12px", border: "1px solid #ddd", borderRadius: 6,
            background: "#fff", fontSize: 13,
          }}>
            <option value="">Tümü</option>
            <option value="online">Online</option>
            <option value="offline">Offline</option>
          </select>
          <input type="text" placeholder="Cihaz ara..." value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ padding: "8px 12px", border: "1px solid #ddd", borderRadius: 6, width: 240, fontSize: 13 }}
          />
        </div>
      </div>

      {filtered.length === 0 ? (
        <div style={{ textAlign: "center", padding: 60, background: "#fff", borderRadius: 8, color: "#999" }}>
          Cihaz bulunamadı. Agent heartbeat göndermeyi bekliyor...
        </div>
      ) : (
        <div style={{ background: "#fff", borderRadius: 8, overflow: "hidden", boxShadow: "0 1px 3px rgba(0,0,0,0.08)" }}>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ background: "#fafafa", textAlign: "left" }}>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>Cihaz</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>Durum</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>CPU</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>RAM</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>Disk</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>POS</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>MSSQL</th>
                <th style={{ padding: "12px 16px", borderBottom: "1px solid #eee" }}>RustDesk</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(d => (
                <tr key={d.id} style={{ borderBottom: "1px solid #f0f0f0", transition: "background 0.2s" }}
                  onMouseEnter={e => (e.currentTarget.style.background = "#fafafa")}
                  onMouseLeave={e => (e.currentTarget.style.background = "transparent")}
                >
                  <td style={{ padding: "12px 16px" }}>
                    <Link to={`/devices/${d.id}`} style={{ fontWeight: 600, color: "#1890ff", textDecoration: "none" }}>
                      {d.hostname}
                    </Link>
                    <div style={{ fontSize: 11, color: "#999", marginTop: 2 }}>{d.os_version || "N/A"}</div>
                  </td>
                  <td style={{ padding: "12px 16px" }}>
                    <span style={{
                      display: "inline-flex", alignItems: "center", gap: 6,
                      padding: "3px 10px", borderRadius: 12, fontSize: 12,
                      background: d.is_online ? "#e6f7e6" : "#ffe6e6",
                      color: d.is_online ? "#52c41a" : "#ff4d4f",
                    }}>
                      <span style={{ width: 6, height: 6, borderRadius: 3, background: d.is_online ? "#52c41a" : "#ff4d4f", display: "inline-block" }} />
                      {d.is_online ? "Online" : "Offline"}
                    </span>
                  </td>
                  <td style={{ padding: "12px 16px" }}>
                    <MiniBar value={d.cpu_percent} color={d.cpu_percent > 80 ? "#ff4d4f" : "#52c41a"} />
                  </td>
                  <td style={{ padding: "12px 16px" }}>
                    <MiniBar value={d.ram_percent} color={d.ram_percent > 80 ? "#ff4d4f" : "#52c41a"} />
                  </td>
                  <td style={{ padding: "12px 16px" }}>
                    <MiniBar value={d.disk_percent} color={d.disk_percent > 80 ? "#ff4d4f" : "#52c41a"} />
                  </td>
                  <td style={{ padding: "12px 16px" }}>
                    <StatusBadge status={d.pos_process_status} />
                  </td>
                  <td style={{ padding: "12px 16px" }}>
                    <StatusBadge status={d.mssql_status} />
                  </td>
                  <td style={{ padding: "12px 16px", fontSize: 11, color: "#666" }}>
                    {d.rustdesk_id || "-"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function MiniBar({ value, color }: { value: number; color: string }) {
  const v = value ?? 0;
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <div style={{ flex: 1, height: 6, background: "#f0f0f0", borderRadius: 3, overflow: "hidden" }}>
        <div style={{ width: `${Math.min(v, 100)}%`, height: "100%", background: color, borderRadius: 3, transition: "width 0.5s" }} />
      </div>
      <span style={{ fontSize: 12, fontWeight: 600, width: 40, textAlign: "right" }}>{v.toFixed(1)}%</span>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const s = (status || "unknown").toLowerCase();
  const color = s === "running" || s === "true" || s === "1" ? "#52c41a" :
                s === "unknown" ? "#999" : "#ff4d4f";
  return <span style={{ color, fontSize: 12, fontWeight: 500 }}>{s}</span>;
}
