import { useState, useEffect } from 'react';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

function getQueryParams(range, customStart, customEnd) {
  const now = new Date();
  if (range === '7d') {
    const start = new Date(now);
    start.setDate(start.getDate() - 7);
    return { start_date: start.toISOString().split('T')[0], end_date: now.toISOString().split('T')[0] };
  }
  if (range === '30d') {
    const start = new Date(now);
    start.setDate(start.getDate() - 30);
    return { start_date: start.toISOString().split('T')[0], end_date: now.toISOString().split('T')[0] };
  }
  if (range === 'custom' && customStart && customEnd) {
    return { start_date: customStart, end_date: customEnd };
  }
  return {};
}

function buildQuery(params) {
  const q = new URLSearchParams(params).toString();
  return q ? `?${q}` : '';
}

function buildChartData(messages, tickets, qa, voice) {
  const data = {};
  for (const row of messages) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, messages: 0, tickets: 0, qa: 0, voice: 0 };
    data[row.member_id].messages += row.count ?? 0;
  }
  for (const row of tickets) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, messages: 0, tickets: 0, qa: 0, voice: 0 };
    data[row.member_id].tickets += row.tickets ?? 0;
  }
  for (const row of qa) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, messages: 0, tickets: 0, qa: 0, voice: 0 };
    data[row.member_id].qa += row.count ?? 0;
  }
  for (const row of voice) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, messages: 0, tickets: 0, qa: 0, voice: 0 };
    data[row.member_id].voice += Math.round((row.total_seconds ?? 0) / 3600);
  }
  return Object.values(data);
}

export default function TeamOverview() {
  const { user } = useAuth();
  const isLoggedIn = user !== null && user !== undefined;
  const isManagement = user?.role === 'manager' || user?.role === 'director';

  const [range, setRange] = useState('30d');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');

  const [chartData, setChartData] = useState([]);
  const [totals, setTotals] = useState({ messages: 0, tickets: 0, qa: 0, voice: 0 });
  const [loading, setLoading] = useState(true);

  const [profiles, setProfiles] = useState({});
  const [selectedModID, setSelectedModID] = useState('');

  useEffect(() => {
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const q = buildQuery(getQueryParams(range, customStart, customEnd));
    setLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${q}`).then(r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice]) => {
        const data = buildChartData(
          Array.isArray(messages) ? messages : [],
          Array.isArray(tickets) ? tickets : [],
          Array.isArray(qa) ? qa : [],
          Array.isArray(voice) ? voice : [],
        );
        setChartData(data);
        setTotals({
          messages: data.reduce((s, d) => s + d.messages, 0),
          tickets: data.reduce((s, d) => s + d.tickets, 0),
          qa: data.reduce((s, d) => s + d.qa, 0),
          voice: data.reduce((s, d) => s + d.voice, 0),
        });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [range, customStart, customEnd]);

  useEffect(() => {
    if (!isManagement || chartData.length === 0) return;
    const ids = chartData.map(d => d.member_id).filter(Boolean);
    fetch(`${BASE}/api/profiles?ids=${ids.join(',')}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => {
        const map = {};
        for (const p of list) map[p.id] = p;
        setProfiles(map);
      })
      .catch(() => {});
  }, [chartData, isManagement]);

  function displayName(id) {
    return profiles[id]?.username || id;
  }

  const selectedModData = selectedModID ? chartData.find(d => d.member_id === selectedModID) : null;
  const displayStats = selectedModData ?? totals;
  const chartDisplayData = selectedModData
    ? [selectedModData]
    : [{ member_id: '__team__', ...totals }];

  const presets = [
    { key: '7d', label: 'Last 7 days' },
    { key: '30d', label: 'Last 30 days' },
    { key: 'all', label: 'All time' },
    { key: 'custom', label: 'Custom' },
  ];

  return (
    <section className="section">
      <h2 className="section-title">Team Overview</h2>

      <div className="date-range-btns">
        {presets.map(({ key, label }) => (
          <button
            key={key}
            className={`date-range-btn${range === key ? ' active' : ''}`}
            onClick={() => { setRange(key); setSelectedModID(''); }}
          >
            {label}
          </button>
        ))}
      </div>

      {range === 'custom' && (
        <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginBottom: 20, flexWrap: 'wrap' }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label" htmlFor="ov-start">From</label>
            <input
              id="ov-start"
              type="date"
              className="form-input"
              style={{ padding: '6px 10px', fontSize: 13 }}
              value={customStart}
              onChange={e => setCustomStart(e.target.value)}
            />
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label" htmlFor="ov-end">To</label>
            <input
              id="ov-end"
              type="date"
              className="form-input"
              style={{ padding: '6px 10px', fontSize: 13 }}
              value={customEnd}
              onChange={e => setCustomEnd(e.target.value)}
            />
          </div>
          {(!customStart || !customEnd) && (
            <span style={{ fontSize: 13, color: 'var(--discord-muted)', alignSelf: 'flex-end', marginBottom: 2 }}>
              Select both dates to load data
            </span>
          )}
        </div>
      )}

      {isManagement && !loading && chartData.length > 0 && (
        <div style={{ marginBottom: 16 }}>
          <select
            className="form-select"
            value={selectedModID}
            onChange={e => setSelectedModID(e.target.value)}
          >
            <option value="">— All Mods (Team Total) —</option>
            {chartData.map(d => (
              <option key={d.member_id} value={d.member_id}>
                {displayName(d.member_id)}
              </option>
            ))}
          </select>
        </div>
      )}

      {selectedModData && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
          {profiles[selectedModID]?.avatar_url && (
            <img
              src={profiles[selectedModID].avatar_url}
              alt=""
              className="user-avatar"
              style={{ width: 28, height: 28 }}
            />
          )}
          <span style={{ fontSize: 13, color: 'var(--discord-muted)' }}>
            Viewing: {displayName(selectedModID)}
          </span>
        </div>
      )}

      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#7289da' }}>
            {loading ? '—' : displayStats.messages.toLocaleString()}
          </div>
          <div className="stat-card-label">Messages</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#43b581' }}>
            {loading ? '—' : displayStats.tickets.toLocaleString()}
          </div>
          <div className="stat-card-label">Tickets</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#faa61a' }}>
            {loading ? '—' : displayStats.qa.toLocaleString()}
          </div>
          <div className="stat-card-label">Q&amp;A</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#f04747' }}>
            {loading ? '—' : `${displayStats.voice}h`}
          </div>
          <div className="stat-card-label">Voice Hours</div>
        </div>
      </div>

      {isLoggedIn && (
        <div className="chart-container">
          {loading ? (
            <p className="loading-text">Loading chart...</p>
          ) : range === 'custom' && (!customStart || !customEnd) ? (
            <p className="loading-text">Select a date range above to view the chart.</p>
          ) : chartDisplayData.length === 0 ? (
            <p className="loading-text">No data for this period.</p>
          ) : (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartDisplayData} margin={{ top: 8, right: 16, left: 0, bottom: 8 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2f3136" />
                <XAxis dataKey="member_id" tick={false} axisLine={false} />
                <YAxis tick={{ fill: '#72767d', fontSize: 11 }} />
                <Tooltip
                  contentStyle={{ background: '#2c2f33', border: '1px solid #2f3136', color: '#dcddde' }}
                  labelFormatter={id => id === '__team__' ? 'Team Total' : displayName(id)}
                />
                <Legend wrapperStyle={{ color: '#dcddde', paddingTop: 8 }} />
                <Bar dataKey="messages" name="Messages" fill="#7289da" />
                <Bar dataKey="tickets" name="Tickets" fill="#43b581" />
                <Bar dataKey="qa" name="Q&A" fill="#faa61a" />
                <Bar dataKey="voice" name="Voice (h)" fill="#f04747" />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      )}
    </section>
  );
}
