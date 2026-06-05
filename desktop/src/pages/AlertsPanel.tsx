import { useState, useEffect } from "react";
import { useAuth, API_BASE } from "../App";

interface Alert {
  id: string; device_id: string; device_name: string; type: string;
  severity: string; message: string; status: string; metric_value: number;
  threshold_value: number; created_at: string; acknowledged_at: string;
  resolved_at: string; title: string;
}

const severityColors: Record<string, string> = {
  critical: "#ff4d4f", warning: "#fa8c16", info: "#1890ff",
};

const statusColors: Record<string, string> = {
  open: "#ff4d4f", acknowledged: "#fa8c16", resolved: "#52c41a",
};

export default function AlertsPanel() {
  const { token } = useAuth();
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [filter, setFilter] = useState("");
  const [showTicketModal, setShowTicketModal] = useState<string | null>(null);
  const [ticketTitle, setTicketTitle] = useState("");

  const load = () => {
    const params = filter ? `?status=${filter}` : "";
    fetch(API_BASE + "/alerts" + params, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(d => setAlerts(d.alerts || []))
      .catch(() => {});
  };

  useEffect(() => { load(); }, [token, filter]);

  const updateStatus = async (id: string, action: string) => {
    await fetch(API_BASE + `/alerts/${id}/${action}`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    });
    load();
  };

  const ackAlert = (id: string) => updateStatus(id, "acknowledge");
  const resolveAlert = (id: string) => updateStatus(id, "resolve");

  const createTicketFromAlert = async (alertId: string) => {
    const alert = alerts.find(a => a.id === alertId);
    if (!alert) return;
    await fetch(API_BASE + "/tickets", {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        title: ticketTitle || `Alert: ${alert.title || alert.type}`,
        description: alert.message || alert.title,
        device_id: alert.device_id,
        alert_id: alert.id,
        source: "alert",
      }),
    });
    setShowTicketModal(null);
    setTicketTitle("");
  };

  const totalOpen = alerts.filter(a => a.status === "open").length;
  const totalCritical = alerts.filter(a => a.severity === "critical").length;

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 24 }}>Alerts</h1>
          <span style={{ fontSize: 13, color: "#666" }}>
            {totalOpen} açık, {totalCritical} kritik
          </span>
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button onClick={load} style={{
            padding: "8px 16px", background: "#1890ff", color: "#fff",
            border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13,
          }}>⟳</button>
          <select value={filter} onChange={e => setFilter(e.target.value)}
            style={{ padding: "8px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 13 }}
          >
            <option value="">Tümü</option>
            <option value="open">Open</option>
            <option value="acknowledged">Acknowledged</option>
            <option value="resolved">Resolved</option>
          </select>
        </div>
      </div>

      {alerts.length === 0 ? (
        <div style={{ textAlign: "center", padding: 60, background: "#fff", borderRadius: 8, color: "#999" }}>
          Alarm bulunamadı.
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {alerts.map(a => (
            <div key={a.id} style={{
              background: "#fff", padding: "16px 20px", borderRadius: 8,
              boxShadow: "0 1px 3px rgba(0,0,0,0.08)",
              borderLeft: `4px solid ${severityColors[a.severity] || "#999"}`,
            }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                <div>
                  <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4 }}>
                    <strong>{a.title || a.type}</strong>
                    <span style={{
                      padding: "2px 8px", borderRadius: 10, fontSize: 11, fontWeight: 600,
                      background: `${statusColors[a.status]}15`,
                      color: statusColors[a.status],
                    }}>{a.status}</span>
                  </div>
                  <p style={{ margin: "2px 0", fontSize: 13, color: "#333" }}>{a.message}</p>
                  <div style={{ fontSize: 12, color: "#999", marginTop: 4 }}>
                    {a.device_name} · {a.severity} · {new Date(a.created_at).toLocaleString("tr-TR")}
                  </div>
                  {a.metric_value != null && (
                    <div style={{ fontSize: 12, color: "#666", marginTop: 2 }}>
                      Değer: {a.metric_value?.toFixed(1)}% / Eşik: {a.threshold_value}%
                    </div>
                  )}
                </div>
                <div style={{ display: "flex", gap: 6 }}>
                  <button onClick={() => { setShowTicketModal(a.id); setTicketTitle(`Alert: ${a.title || a.type}`); }} style={{
                    padding: "5px 12px", background: "#722ed1", color: "#fff",
                    border: "none", borderRadius: 4, cursor: "pointer", fontSize: 12,
                  }}>Ticket Oluştur</button>
                  {a.status === "open" && (
                    <button onClick={() => ackAlert(a.id)} style={{
                      padding: "5px 12px", background: "#fa8c16", color: "#fff",
                      border: "none", borderRadius: 4, cursor: "pointer", fontSize: 12,
                    }}>Acknowledge</button>
                  )}
                  {(a.status === "open" || a.status === "acknowledged") && (
                    <button onClick={() => resolveAlert(a.id)} style={{
                      padding: "5px 12px", background: "#52c41a", color: "#fff",
                      border: "none", borderRadius: 4, cursor: "pointer", fontSize: 12,
                    }}>Resolve</button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showTicketModal && (
        <div style={{
          position: "fixed", top: 0, left: 0, right: 0, bottom: 0,
          background: "rgba(0,0,0,0.5)", display: "flex",
          justifyContent: "center", alignItems: "center", zIndex: 1000,
        }} onClick={() => setShowTicketModal(null)}>
          <div style={{
            background: "#fff", padding: 24, borderRadius: 12, width: 440,
          }} onClick={e => e.stopPropagation()}>
            <h2 style={{ margin: "0 0 16px", fontSize: 18 }}>Ticket Oluştur</h2>
            <div style={{ marginBottom: 16 }}>
              <label style={{ display: "block", marginBottom: 6, fontSize: 13 }}>Başlık</label>
              <input value={ticketTitle} onChange={e => setTicketTitle(e.target.value)}
                style={{ width: "100%", padding: "10px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 14 }}
              />
            </div>
            <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
              <button onClick={() => setShowTicketModal(null)} style={{
                padding: "8px 16px", background: "#f0f0f0", border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13,
              }}>İptal</button>
              <button onClick={() => createTicketFromAlert(showTicketModal)} style={{
                padding: "8px 16px", background: "#722ed1", color: "#fff", border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13,
              }}>Oluştur</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
