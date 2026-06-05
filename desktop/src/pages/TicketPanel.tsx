import { useState, useEffect } from "react";
import { useAuth, API_BASE } from "../App";

interface Ticket {
  id: string; title: string; description: string; status: string;
  priority: string; device_id: string; device_name: string;
  assigned_to: string; assigned_name: string;
  created_by: string; created_name: string;
  alert_id: string; ticket_number: number;
  created_at: string; updated_at: string;
}

const statusColors: Record<string, string> = {
  open: "#ff4d4f", in_progress: "#1890ff", resolved: "#52c41a", closed: "#999",
};

export default function TicketPanel() {
  const { token } = useAuth();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [devices, setDevices] = useState<any[]>([]);
  const [filter, setFilter] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ title: "", description: "", priority: "medium", device_id: "" });

  const load = () => {
    const params = filter ? `?status=${filter}` : "";
    fetch(API_BASE + "/tickets" + params, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(d => setTickets(d.tickets || []))
      .catch(() => {});
    fetch(API_BASE + "/devices?limit=100", {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(d => setDevices(d.devices || []))
      .catch(() => {});
  };

  useEffect(() => { load(); }, [token, filter]);

  const updateStatus = async (id: string, status: string) => {
    await fetch(API_BASE + `/tickets/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ status }),
    });
    load();
  };

  const createTicket = async () => {
    if (!form.title) return;
    await fetch(API_BASE + "/tickets", {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify(form),
    });
    setShowCreate(false);
    setForm({ title: "", description: "", priority: "medium", device_id: "" });
    load();
  };

  const totalOpen = tickets.filter(t => t.status === "open").length;

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 24 }}>Tickets</h1>
          <span style={{ fontSize: 13, color: "#666" }}>{totalOpen} açık ticket</span>
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button onClick={() => setShowCreate(true)} style={{
            padding: "8px 16px", background: "#52c41a", color: "#fff",
            border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13, fontWeight: 600,
          }}>+ Yeni Ticket</button>
          <select value={filter} onChange={e => setFilter(e.target.value)}
            style={{ padding: "8px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 13 }}
          >
            <option value="">Tümü</option>
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
        </div>
      </div>

      {tickets.length === 0 ? (
        <div style={{ textAlign: "center", padding: 60, background: "#fff", borderRadius: 8, color: "#999" }}>
          Ticket bulunamadı.
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {tickets.map(t => (
            <div key={t.id} style={{
              background: "#fff", padding: "16px 20px", borderRadius: 8,
              boxShadow: "0 1px 3px rgba(0,0,0,0.08)",
              borderLeft: `4px solid ${statusColors[t.status] || "#999"}`,
            }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                <div>
                  <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4 }}>
                    <strong style={{ fontSize: 14 }}>#{t.ticket_number} {t.title}</strong>
                    <span style={{
                      padding: "2px 8px", borderRadius: 10, fontSize: 11, fontWeight: 600,
                      background: `${statusColors[t.status]}15`,
                      color: statusColors[t.status],
                    }}>{t.status}</span>
                  </div>
                  <p style={{ margin: "4px 0", fontSize: 13, color: "#555" }}>{t.description}</p>
                  <div style={{ fontSize: 12, color: "#999", marginTop: 4 }}>
                    {t.device_name && <span>{t.device_name} · </span>}
                    {t.created_name && <span>{t.created_name} · </span>}
                    {t.created_at && <>Oluşturulma: {new Date(t.created_at).toLocaleString("tr-TR")}</>}
                  </div>
                </div>
                <div style={{ display: "flex", gap: 6 }}>
                  {t.status === "open" && (
                    <button onClick={() => updateStatus(t.id, "in_progress")} style={{
                      padding: "5px 12px", background: "#1890ff", color: "#fff",
                      border: "none", borderRadius: 4, cursor: "pointer", fontSize: 12,
                    }}>Başla</button>
                  )}
                  {(t.status === "open" || t.status === "in_progress") && (
                    <button onClick={() => updateStatus(t.id, "resolved")} style={{
                      padding: "5px 12px", background: "#52c41a", color: "#fff",
                      border: "none", borderRadius: 4, cursor: "pointer", fontSize: 12,
                    }}>Çöz</button>
                  )}
                  {t.status === "resolved" && (
                    <button onClick={() => updateStatus(t.id, "closed")} style={{
                      padding: "5px 12px", background: "#999", color: "#fff",
                      border: "none", borderRadius: 4, cursor: "pointer", fontSize: 12,
                    }}>Kapat</button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreate && (
        <div style={{
          position: "fixed", top: 0, left: 0, right: 0, bottom: 0,
          background: "rgba(0,0,0,0.5)", display: "flex",
          justifyContent: "center", alignItems: "center", zIndex: 1000,
        }} onClick={() => setShowCreate(false)}>
          <div style={{
            background: "#fff", padding: 24, borderRadius: 12, width: 480,
          }} onClick={e => e.stopPropagation()}>
            <h2 style={{ margin: "0 0 16px", fontSize: 18 }}>Yeni Ticket</h2>
            <div style={{ marginBottom: 12 }}>
              <label style={{ display: "block", marginBottom: 6, fontSize: 13 }}>Başlık *</label>
              <input value={form.title} onChange={e => setForm({...form, title: e.target.value})}
                style={{ width: "100%", padding: "10px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 14 }}
              />
            </div>
            <div style={{ marginBottom: 12 }}>
              <label style={{ display: "block", marginBottom: 6, fontSize: 13 }}>Açıklama</label>
              <textarea value={form.description} onChange={e => setForm({...form, description: e.target.value})}
                rows={3}
                style={{ width: "100%", padding: "10px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 14, resize: "vertical" }}
              />
            </div>
            <div style={{ display: "flex", gap: 12, marginBottom: 16 }}>
              <div style={{ flex: 1 }}>
                <label style={{ display: "block", marginBottom: 6, fontSize: 13 }}>Öncelik</label>
                <select value={form.priority} onChange={e => setForm({...form, priority: e.target.value})}
                  style={{ width: "100%", padding: "10px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 14 }}
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </div>
              <div style={{ flex: 1 }}>
                <label style={{ display: "block", marginBottom: 6, fontSize: 13 }}>Cihaz</label>
                <select value={form.device_id} onChange={e => setForm({...form, device_id: e.target.value})}
                  style={{ width: "100%", padding: "10px 12px", border: "1px solid #ddd", borderRadius: 6, fontSize: 14 }}
                >
                  <option value="">Seçilmedi</option>
                  {devices.map(d => <option key={d.id} value={d.id}>{d.hostname}</option>)}
                </select>
              </div>
            </div>
            <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
              <button onClick={() => setShowCreate(false)} style={{
                padding: "8px 16px", background: "#f0f0f0", border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13,
              }}>İptal</button>
              <button onClick={createTicket} style={{
                padding: "8px 16px", background: "#52c41a", color: "#fff", border: "none", borderRadius: 6, cursor: "pointer", fontSize: 13,
              }}>Oluştur</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
