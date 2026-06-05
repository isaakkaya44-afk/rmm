import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { useAuth, API_BASE } from "../App";

interface Device {
  id: string; hostname: string; os_version: string; cpu_model: string;
  total_ram_gb: number; total_disk_gb: number; is_online: boolean;
  cpu_percent: number; ram_percent: number; disk_percent: number;
  last_heartbeat: string; pos_process_status: string; mssql_status: string;
  rustdesk_id: string; rustdesk_password: string; agent_version: string;
  tags: string;
}

export default function DeviceDetail() {
  const { id } = useParams();
  const { token } = useAuth();
  const [dev, setDev] = useState<Device | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    fetch(API_BASE + `/devices/${id}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(d => setDev(d))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [id, token]);

  if (loading) return <div style={{ textAlign: "center", padding: 60, color: "#999" }}>Yükleniyor...</div>;
  if (!dev) return <div style={{ textAlign: "center", padding: 60, color: "#999" }}>Cihaz bulunamadı.</div>;

  const metrics = [
    { label: "CPU", value: dev.cpu_percent, warn: 80, total: 100, unit: "%" },
    { label: "RAM", value: dev.ram_percent, warn: 80, total: dev.total_ram_gb, unit: "%" },
    { label: "Disk", value: dev.disk_percent, warn: 80, total: dev.total_disk_gb, unit: "%" },
  ];

  return (
    <div>
      <div style={{ marginBottom: 20 }}>
        <Link to="/" style={{ color: "#1890ff", textDecoration: "none", fontSize: 13 }}>← Dashboard</Link>
      </div>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 24 }}>{dev.hostname}</h1>
          <p style={{ margin: "4px 0 0", color: "#666", fontSize: 13 }}>{dev.os_version}</p>
        </div>
        <span style={{
          padding: "6px 14px", borderRadius: 16, fontSize: 13, fontWeight: 600,
          background: dev.is_online ? "#e6f7e6" : "#ffe6e6",
          color: dev.is_online ? "#52c41a" : "#ff4d4f",
        }}>{dev.is_online ? "Online" : "Offline"}</span>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 24 }}>
        <div style={{ background: "#fff", borderRadius: 8, padding: 20, boxShadow: "0 1px 3px rgba(0,0,0,0.08)" }}>
          <h3 style={{ margin: "0 0 16px", fontSize: 15 }}>Sistem Metrikleri</h3>
          {metrics.map(m => (
            <div key={m.label} style={{ marginBottom: 16 }}>
              <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 4, fontSize: 13 }}>
                <span>{m.label}</span>
                <span style={{ fontWeight: 600 }}>{(m.value ?? 0).toFixed(1)}{m.unit}</span>
              </div>
              <div style={{ height: 10, background: "#f0f0f0", borderRadius: 5, overflow: "hidden" }}>
                <div style={{
                  width: `${Math.min(m.value ?? 0, 100)}%`, height: "100%",
                  background: (m.value ?? 0) > m.warn ? "#ff4d4f" : (m.value ?? 0) > 60 ? "#fa8c16" : "#52c41a",
                  borderRadius: 5, transition: "width 0.5s",
                }} />
              </div>
            </div>
          ))}
        </div>

        <div style={{ background: "#fff", borderRadius: 8, padding: 20, boxShadow: "0 1px 3px rgba(0,0,0,0.08)" }}>
          <h3 style={{ margin: "0 0 16px", fontSize: 15 }}>Servis Durumu</h3>
          <div style={{ display: "grid", gap: 12 }}>
            <ServiceRow label="POS Process" status={dev.pos_process_status} />
            <ServiceRow label="MSSQL" status={dev.mssql_status} />
            <ServiceRow label="Agent" version={dev.agent_version} />
            <ServiceRow label="Last Heartbeat" time={dev.last_heartbeat} />
          </div>
        </div>
      </div>

      {dev.rustdesk_id && (
        <div style={{ marginTop: 24, background: "#fff", borderRadius: 8, padding: 20, boxShadow: "0 1px 3px rgba(0,0,0,0.08)" }}>
          <h3 style={{ margin: "0 0 16px", fontSize: 15 }}>RustDesk Remote</h3>
          <div style={{ display: "flex", gap: 16, alignItems: "center" }}>
            <div>
              <span style={{ fontSize: 12, color: "#666" }}>ID</span>
              <p style={{ margin: "2px 0", fontSize: 16, fontWeight: 700, fontFamily: "monospace" }}>{dev.rustdesk_id}</p>
            </div>
            {dev.rustdesk_password && (
              <div>
                <span style={{ fontSize: 12, color: "#666" }}>Password</span>
                <p style={{ margin: "2px 0", fontSize: 16, fontWeight: 700, fontFamily: "monospace" }}>{dev.rustdesk_password}</p>
              </div>
            )}
            <a href={`rustdesk://${dev.rustdesk_id}`} style={{
              marginLeft: "auto", padding: "10px 20px", background: "#1890ff", color: "#fff",
              textDecoration: "none", borderRadius: 6, fontWeight: 600, fontSize: 13,
            }}>🔗 Bağlan</a>
          </div>
        </div>
      )}
    </div>
  );
}

function ServiceRow({ label, status, version, time }: { label: string; status?: string; version?: string; time?: string }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", padding: "8px 0", borderBottom: "1px solid #f0f0f0" }}>
      <span style={{ fontSize: 13, color: "#666" }}>{label}</span>
      <span style={{ fontSize: 13, fontWeight: 500 }}>
        {status && (
          <span style={{
            color: status === "running" || status === "true" || status === "1" ? "#52c41a" :
                  status === "unknown" ? "#999" : "#ff4d4f",
          }}>{status}</span>
        )}
        {version && <span style={{ color: "#666" }}>{version}</span>}
        {time && <span style={{ color: "#999" }}>{new Date(time).toLocaleString("tr-TR")}</span>}
      </span>
    </div>
  );
}
