import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import ModNotesTab from './ModNotesTab';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const TABS = ['Messages', 'Tickets', 'Q&A', 'Voice Hours', 'Mod Notes'];
const METRIC_KEY = { Messages: 'messages', Tickets: 'tickets', 'Q&A': 'qa', 'Voice Hours': 'voice' };

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

function buildData(messages, tickets, qa, voice) {
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

export default function ManagementView() {
  const { user } = useAuth();
  const isDirector = user?.role === 'director';

  const [activeTab, setActiveTab] = useState('Messages');
  const [range, setRange] = useState('30d');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [profiles, setProfiles] = useState({});
  const [sortDir, setSortDir] = useState('desc');

  const presets = [
    { key: '7d', label: 'Last 7 days' },
    { key: '30d', label: 'Last 30 days' },
    { key: 'all', label: 'All time' },
    { key: 'custom', label: 'Custom' },
  ];

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
        setData(buildData(
          Array.isArray(messages) ? messages : [],
          Array.isArray(tickets) ? tickets : [],
          Array.isArray(qa) ? qa : [],
          Array.isArray(voice) ? voice : [],
        ));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [range, customStart, customEnd]);

  useEffect(() => {
    if (data.length === 0) return;
    const ids = data.map(d => d.member_id).filter(Boolean);
    fetch(`${BASE}/api/profiles?ids=${ids.join(',')}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => {
        const map = {};
        for (const p of list) map[p.id] = p;
        setProfiles(prev => ({ ...prev, ...map }));
      })
      .catch(() => {});
  }, [data]);

  function displayName(id) {
    return profiles[id]?.username || id;
  }

  const metricKey = METRIC_KEY[activeTab];
  const sortedData = [...data].sort((a, b) => {
    const diff = (b[metricKey] ?? 0) - (a[metricKey] ?? 0);
    return sortDir === 'desc' ? diff : -diff;
  });

  function formatValue(row) {
    if (activeTab === 'Voice Hours') return `${row.voice ?? 0}h`;
    return (row[metricKey] ?? 0).toLocaleString();
  }

  return (
    <section className="section">
      <h2 className="section-title">Management</h2>

      <div className="tab-nav">
        {TABS.map(tab => (
          <button
            key={tab}
            className={`tab-btn${activeTab === tab ? ' active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'Mod Notes' ? (
        <ModNotesTab
          isDirector={isDirector}
          modIds={data.map(d => d.member_id)}
          profiles={profiles}
          setProfiles={setProfiles}
        />
      ) : (
        <>
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
                <label className="form-label">From</label>
                <input
                  type="date"
                  className="form-input"
                  style={{ padding: '6px 10px', fontSize: 13 }}
                  value={customStart}
                  onChange={e => setCustomStart(e.target.value)}
                />
              </div>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">To</label>
                <input
                  type="date"
                  className="form-input"
                  style={{ padding: '6px 10px', fontSize: 13 }}
                  value={customEnd}
                  onChange={e => setCustomEnd(e.target.value)}
                />
              </div>
            </div>
          )}

          {loading ? (
            <p className="loading-text">Loading...</p>
          ) : range === 'custom' && (!customStart || !customEnd) ? (
            <p className="loading-text">Select a date range to view data.</p>
          ) : sortedData.length === 0 ? (
            <p className="loading-text">No data for this period.</p>
          ) : (
            <table className="loa-table">
              <thead>
                <tr>
                  <th>Mod</th>
                  <th
                    style={{ cursor: 'pointer', userSelect: 'none' }}
                    onClick={() => setSortDir(d => d === 'desc' ? 'asc' : 'desc')}
                  >
                    {activeTab} {sortDir === 'desc' ? '▼' : '▲'}
                  </th>
                </tr>
              </thead>
              <tbody>
                {sortedData.map(row => (
                  <tr key={row.member_id}>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        {profiles[row.member_id]?.avatar_url && (
                          <img
                            src={profiles[row.member_id].avatar_url}
                            alt=""
                            className="user-avatar"
                            style={{ width: 20, height: 20 }}
                          />
                        )}
                        {displayName(row.member_id)}
                      </div>
                    </td>
                    <td>{formatValue(row)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </section>
  );
}
