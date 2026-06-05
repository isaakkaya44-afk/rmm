import { Outlet, Link, useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../App";

interface LayoutProps {
  wsEvents: any[];
}

export default function Layout({ wsEvents }: LayoutProps) {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const links = [
    { to: "/", label: "Dashboard", icon: "📊" },
    { to: "/alerts", label: "Alerts", icon: "🔔" },
    { to: "/tickets", label: "Tickets", icon: "🎫" },
  ];

  return (
    <div style={{ display: "flex", minHeight: "100vh", fontFamily: "'Segoe UI', sans-serif" }}>
      <nav style={{
        width: 240,
        background: "linear-gradient(180deg, #1a1a2e 0%, #16213e 100%)",
        color: "#fff",
        padding: 0,
        display: "flex",
        flexDirection: "column",
      }}>
        <div style={{ padding: "24px 20px", borderBottom: "1px solid rgba(255,255,255,0.1)" }}>
          <h2 style={{ margin: 0, fontSize: 18, fontWeight: 700 }}>RMM Panel</h2>
          <p style={{ fontSize: 12, opacity: 0.6, margin: "8px 0 0" }}>
            {user?.full_name}
            <span style={{ display: "inline-block", marginLeft: 8, padding: "2px 8px", background: "rgba(255,255,255,0.1)", borderRadius: 10, fontSize: 10 }}>
              {user?.role}
            </span>
          </p>
        </div>

        <ul style={{ listStyle: "none", padding: "16px 12px", flex: 1 }}>
          {links.map((link) => (
            <li key={link.to} style={{ marginBottom: 4 }}>
              <Link
                to={link.to}
                style={{
                  color: "#fff",
                  textDecoration: "none",
                  display: "flex",
                  alignItems: "center",
                  gap: 10,
                  padding: "10px 12px",
                  borderRadius: 8,
                  background: location.pathname === link.to ? "rgba(255,255,255,0.1)" : "transparent",
                  fontSize: 14,
                  transition: "background 0.2s",
                }}
              >
                <span>{link.icon}</span>
                {link.label}
              </Link>
            </li>
          ))}
        </ul>

        <div style={{ padding: "16px 16px" }}>
          <div style={{ marginBottom: 12, fontSize: 12, opacity: 0.5 }}>
            Son olaylar:
          </div>
          <div style={{ maxHeight: 150, overflow: "auto", fontSize: 11, opacity: 0.7 }}>
            {wsEvents.slice(0, 5).map((evt, i) => (
              <div key={i} style={{ padding: "4px 0", borderBottom: "1px solid rgba(255,255,255,0.05)" }}>
                {evt.type}
              </div>
            ))}
            {wsEvents.length === 0 && <div style={{ opacity: 0.4 }}>Bekleniyor...</div>}
          </div>
        </div>

        <button
          onClick={handleLogout}
          style={{
            margin: "0 16px 16px",
            padding: "10px 16px",
            background: "transparent",
            border: "1px solid rgba(255,255,255,0.3)",
            color: "#fff",
            borderRadius: 8,
            cursor: "pointer",
            fontSize: 13,
          }}
        >
          Çıkış Yap
        </button>
      </nav>
      <main style={{ flex: 1, padding: 24, background: "#f0f2f5", overflow: "auto" }}>
        <Outlet />
      </main>
    </div>
  );
}
