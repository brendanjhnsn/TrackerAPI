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
  if (range === '90d') {
    const start = new Date(now);
    start.setDate(start.getDate() - 90);
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
  const [viewMode, setViewMode] = useState('team');

  const [chartData, setChartData] = useState([]);
  const [totals, setTotals] = useState({ messages: 0, tickets: 0, qa: 0, voice: 0 });
  const [loading, setLoading] = useState(true);

  const [myStats, setMyStats] = useState({ messages: 0, tickets: 0, qa: 0, voice: 0 });
  const [myLoading, setMyLoading] = useState(false);

  useEffect(() => {
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const q = buildQuery(getQueryParams(range, customStart, customEnd));
    setLoading(true);
    const removedFetch = isManagement
      ? fetch(`${BASE}/api/removed-mods`, { credentials: 'include' }).then(r => r.ok ? r.json() : [])
      : Promise.resolve([]);
    Promise.all([
      fetch(`${BASE}/api/messages${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${q}`).then(r => r.ok ? r.json() : []),
      removedFetch,
    ])
      .then(([messages, tickets, qa, voice, removed]) => {
        const removedIDs = new Set((Array.isArray(removed) ? removed : []).map(r => r.MemberID));
        const exclude = id => !removedIDs.has(id);
        const data = buildChartData(
          (Array.isArray(messages) ? messages : []).filter(r => exclude(r.member_id)),
          (Array.isArray(tickets)  ? tickets  : []).filter(r => exclude(r.member_id)),
          (Array.isArray(qa)       ? qa       : []).filter(r => exclude(r.member_id)),
          (Array.isArray(voice)    ? voice    : []).filter(r => exclude(r.member_id)),
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
  }, [range, customStart, customEnd, isManagement]);

  useEffect(() => {
    if (!isLoggedIn || viewMode !== 'mine' || !user) return;
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const memberID = user.discord_user_id;
    const params = getQueryParams(range, customStart, customEnd);
    const myQ = buildQuery({ ...params, member_id: memberID });
    setMyLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages${myQ}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${myQ}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${myQ}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${myQ}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice]) => {
        setMyStats({
          messages: Array.isArray(messages) ? messages.reduce((s, r) => s + (r.count ?? 0), 0) : 0,
          tickets:  Array.isArray(tickets)  ? tickets.reduce((s, r) => s + (r.tickets ?? 0), 0) : 0,
          qa:       Array.isArray(qa)       ? qa.reduce((s, r) => s + (r.count ?? 0), 0) : 0,
          voice:    Array.isArray(voice)    ? Math.round(voice.reduce((s, r) => s + (r.total_seconds ?? 0), 0) / 3600) : 0,
        });
      })
      .catch(() => {})
      .finally(() => setMyLoading(false));
  }, [isLoggedIn, viewMode, user, range, customStart, customEnd]);

  const displayStats = viewMode === 'mine' ? myStats : totals;
  const isLoadingDisplay = viewMode === 'mine' ? myLoading : loading;
  const chartDisplayData = viewMode === 'mine' && user
    ? [{ member_id: user.discord_user_id, ...myStats }]
    : [{ member_id: '__team__', ...totals }];

  const presets = [
    { key: '7d', label: 'Last 7 days' },
    { key: '30d', label: 'Last 30 days' },
    { key: '90d', label: 'Last 90 days' },
    { key: 'all', label: 'All time' },
    { key: 'custom', label: 'Custom' },
  ];

  return (
    <section className="section">
      <h2 className="section-title">Team Overview</h2>

      {isLoggedIn && (
        <div className="view-toggle">
          <button
            className={`view-toggle-btn${viewMode === 'team' ? ' active' : ''}`}
            onClick={() => setViewMode('team')}
          >
            Team Stats
          </button>
          <button
            className={`view-toggle-btn${viewMode === 'mine' ? ' active' : ''}`}
            onClick={() => setViewMode('mine')}
          >
            My Stats
          </button>
        </div>
      )}

      <div className="date-range-btns">
        {presets.map(({ key, label }) => (
          <button
            key={key}
            className={`date-range-btn${range === key ? ' active' : ''}`}
            onClick={() => setRange(key)}
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

      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#7289da' }}>
            {isLoadingDisplay ? '—' : displayStats.messages.toLocaleString()}
          </div>
          <div className="stat-card-label">Messages</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#43b581' }}>
            {isLoadingDisplay ? '—' : displayStats.tickets.toLocaleString()}
          </div>
          <div className="stat-card-label">Tickets</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#faa61a' }}>
            {isLoadingDisplay ? '—' : displayStats.qa.toLocaleString()}
          </div>
          <div className="stat-card-label">Q&amp;A</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: '#f04747' }}>
            {isLoadingDisplay ? '—' : `${displayStats.voice}h`}
          </div>
          <div className="stat-card-label">Voice Hours</div>
        </div>
      </div>

      <div className="chart-container">
        {loading ? (
          <p className="loading-text">Loading chart...</p>
        ) : range === 'custom' && (!customStart || !customEnd) ? (
          <p className="loading-text">Select a date range above to view the chart.</p>
        ) : chartDisplayData.length === 0 || (chartDisplayData[0].messages === 0 && chartDisplayData[0].tickets === 0 && chartDisplayData[0].qa === 0 && chartDisplayData[0].voice === 0) ? (
          <p className="loading-text">No data for this period.</p>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartDisplayData} margin={{ top: 8, right: 16, left: 0, bottom: 8 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2f3136" />
              <XAxis dataKey="member_id" tick={false} axisLine={false} />
              <YAxis tick={{ fill: '#72767d', fontSize: 11 }} />
              <Tooltip
                contentStyle={{ background: '#2c2f33', border: '1px solid #2f3136', color: '#dcddde' }}
                labelFormatter={() => viewMode === 'mine' ? (user?.username || 'My Stats') : 'Team Total'}
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
    </section>
  );
}
