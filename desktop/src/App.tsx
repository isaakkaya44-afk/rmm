import { useState, useEffect, createContext, useContext, useRef, useCallback } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import DeviceDetail from "./pages/DeviceDetail";
import TicketPanel from "./pages/TicketPanel";
import AlertsPanel from "./pages/AlertsPanel";
import Layout from "./components/Layout";

interface AuthContextType {
  token: string | null;
  user: any | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

export const AuthContext = createContext<AuthContextType>({
  token: null,
  user: null,
  login: async () => {},
  logout: () => {},
});

export const useAuth = () => useContext(AuthContext);
const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
export const API_BASE = "/api/v1";
export const WS_URL = `${proto}//${window.location.host}/ws`;

function App() {
  const [token, setToken] = useState<string | null>(
    localStorage.getItem("rmm_token")
  );
  const [user, setUser] = useState<any | null>(null);
  const [wsEvents, setWsEvents] = useState<any[]>([]);

  useEffect(() => {
    if (token) {
      fetch(API_BASE + "/me", {
        headers: { Authorization: `Bearer ${token}` },
      })
        .then((r) => r.json())
        .then((u) => setUser(u))
        .catch(() => {
          localStorage.removeItem("rmm_token");
          setToken(null);
        });
    }
  }, [token]);

  const handleWsMessage = useCallback((msg: any) => {
    setWsEvents(prev => [msg, ...prev].slice(0, 50));
  }, []);

  useEffect(() => {
    if (!token) return;
    const ws = new WebSocket(WS_URL);
    ws.onopen = () => console.log("WS connected");
    ws.onmessage = (e) => {
      try { handleWsMessage(JSON.parse(e.data)); } catch {}
    };
    ws.onclose = () => {};
    return () => ws.close();
  }, [token, handleWsMessage]);

  const login = async (email: string, password: string) => {
    const res = await fetch(API_BASE + "/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) throw new Error("Login failed");
    const data = await res.json();
    localStorage.setItem("rmm_token", data.access_token);
    setToken(data.access_token);
    setUser(data.user);
  };

  const logout = () => {
    localStorage.removeItem("rmm_token");
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ token, user, login, logout }}>
      <BrowserRouter>
        <Routes>
          <Route
            path="/login"
            element={token ? <Navigate to="/" /> : <Login />}
          />
          <Route
            path="/"
            element={token ? <Layout wsEvents={wsEvents} /> : <Navigate to="/login" />}
          >
            <Route index element={<Dashboard wsEvents={wsEvents} />} />
            <Route path="devices/:id" element={<DeviceDetail />} />
            <Route path="tickets" element={<TicketPanel />} />
            <Route path="alerts" element={<AlertsPanel />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthContext.Provider>
  );
}

export default App;
