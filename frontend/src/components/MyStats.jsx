import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

export default function MyStats() {
  const { user } = useAuth();
  const [stats, setStats] = useState({ messages: 0, tickets: 0, qa: 0, voice: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) return;
    const memberID = user.discord_user_id;
    setLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages?member_id=${memberID}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets?member_id=${memberID}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs?member_id=${memberID}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice?member_id=${memberID}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice]) => {
        const msgs = Array.isArray(messages) ? messages.reduce((s, r) => s + (r.count ?? 0), 0) : 0;
        const tkts = Array.isArray(tickets) ? tickets.reduce((s, r) => s + (r.tickets ?? 0), 0) : 0;
        const qas = Array.isArray(qa) ? qa.reduce((s, r) => s + (r.count ?? 0), 0) : 0;
        const vhr = Array.isArray(voice) ? Math.round(voice.reduce((s, r) => s + (r.total_seconds ?? 0), 0) / 3600) : 0;
        setStats({ messages: msgs, tickets: tkts, qa: qas, voice: vhr });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [user]);

  return (
    <section className="section">
      <h2 className="section-title">My Stats</h2>
      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#7289da' }}>
            {loading ? '—' : stats.messages.toLocaleString()}
          </div>
          <div className="stat-card-label">Messages</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#43b581' }}>
            {loading ? '—' : stats.tickets.toLocaleString()}
          </div>
          <div className="stat-card-label">Tickets</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#faa61a' }}>
            {loading ? '—' : stats.qa.toLocaleString()}
          </div>
          <div className="stat-card-label">Q&amp;A</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#f04747' }}>
            {loading ? '—' : `${stats.voice}h`}
          </div>
          <div className="stat-card-label">Voice Hours</div>
        </div>
      </div>
    </section>
  );
}
