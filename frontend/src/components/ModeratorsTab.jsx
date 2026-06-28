import { useState, useEffect } from 'react';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const PRESETS = [
  { key: '7d', label: '7 days' },
  { key: '30d', label: '30 days' },
  { key: '90d', label: '90 days' },
  { key: 'all', label: 'All time' },
  { key: 'custom', label: 'Custom' },
];

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

export default function ModeratorsTab({ modIds, profiles, setProfiles, removedIds, onRemoveMod }) {
  const [selectedModID, setSelectedModID] = useState(null);
  const [stats, setStats] = useState(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const [range, setRange] = useState('30d');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');
  const [confirmRemove, setConfirmRemove] = useState(null);

  useEffect(() => {
    if (!selectedModID) return;
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const params = getQueryParams(range, customStart, customEnd);
    const q = buildQuery({ ...params, member_id: selectedModID });
    setStatsLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages${q}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${q}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${q}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${q}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice]) => {
        setStats({
          messages: Array.isArray(messages) ? messages.reduce((s, r) => s + (r.count ?? 0), 0) : 0,
          tickets:  Array.isArray(tickets)  ? tickets.reduce((s, r) => s + (r.tickets ?? 0), 0) : 0,
          qa:       Array.isArray(qa)       ? qa.reduce((s, r) => s + (r.count ?? 0), 0) : 0,
          voice:    Array.isArray(voice)    ? Math.round(voice.reduce((s, r) => s + (r.total_seconds ?? 0), 0) / 3600) : 0,
        });
      })
      .catch(() => {})
      .finally(() => setStatsLoading(false));
  }, [selectedModID, range, customStart, customEnd]);

  function handleBack() {
    setSelectedModID(null);
    setStats(null);
    setConfirmRemove(null);
  }

  function displayName(id) {
    return profiles[id]?.username || id;
  }

  const activeMods = modIds.filter(id => !removedIds.has(id));

  // ---- Detail view ----
  if (selectedModID) {
    return (
      <div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
          <button className="btn btn-muted btn-sm" onClick={handleBack}>← Back</button>
          {profiles[selectedModID]?.avatar_url && (
            <img src={profiles[selectedModID].avatar_url} alt="" className="user-avatar" style={{ width: 28, height: 28 }} />
          )}
          <span style={{ fontSize: 15, fontWeight: 600 }}>{displayName(selectedModID)}</span>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 8, alignItems: 'center' }}>
            {confirmRemove === selectedModID ? (
              <>
                <span style={{ fontSize: 13, color: 'var(--discord-muted)' }}>Remove this mod?</span>
                <button
                  className="btn btn-red btn-sm"
                  onClick={() => { onRemoveMod(selectedModID); handleBack(); }}
                >
                  Confirm
                </button>
                <button className="btn btn-muted btn-sm" onClick={() => setConfirmRemove(null)}>
                  Cancel
                </button>
              </>
            ) : (
              <button className="btn btn-red btn-sm" onClick={() => setConfirmRemove(selectedModID)}>
                Remove Mod
              </button>
            )}
          </div>
        </div>

        <div className="date-range-btns">
          {PRESETS.map(({ key, label }) => (
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
              <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13 }}
                value={customStart} onChange={e => setCustomStart(e.target.value)} />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">To</label>
              <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13 }}
                value={customEnd} onChange={e => setCustomEnd(e.target.value)} />
            </div>
          </div>
        )}

        <div className="stat-cards">
          <div className="stat-card">
            <div className="stat-card-value" style={{ color: '#7289da' }}>
              {statsLoading ? '—' : (stats?.messages ?? 0).toLocaleString()}
            </div>
            <div className="stat-card-label">Messages</div>
          </div>
          <div className="stat-card">
            <div className="stat-card-value" style={{ color: '#43b581' }}>
              {statsLoading ? '—' : (stats?.tickets ?? 0).toLocaleString()}
            </div>
            <div className="stat-card-label">Tickets</div>
          </div>
          <div className="stat-card">
            <div className="stat-card-value" style={{ color: '#faa61a' }}>
              {statsLoading ? '—' : (stats?.qa ?? 0).toLocaleString()}
            </div>
            <div className="stat-card-label">Q&amp;A</div>
          </div>
          <div className="stat-card">
            <div className="stat-card-value" style={{ color: '#f04747' }}>
              {statsLoading ? '—' : `${stats?.voice ?? 0}h`}
            </div>
            <div className="stat-card-label">Voice Hours</div>
          </div>
        </div>
      </div>
    );
  }

  // ---- List view ----
  return (
    <div>
      {activeMods.length === 0 ? (
        <p className="loading-text">No active mods found. Mods appear here once they have activity data.</p>
      ) : (
        <table className="loa-table">
          <thead>
            <tr>
              <th>Mod</th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {activeMods.map(id => (
              <tr key={id}>
                <td>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {profiles[id]?.avatar_url && (
                      <img src={profiles[id].avatar_url} alt="" className="user-avatar" style={{ width: 20, height: 20 }} />
                    )}
                    {displayName(id)}
                  </div>
                </td>
                <td>
                  <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', alignItems: 'center' }}>
                    {confirmRemove === id ? (
                      <>
                        <span style={{ fontSize: 12, color: 'var(--discord-muted)' }}>Sure?</span>
                        <button className="btn btn-red btn-sm" onClick={() => { onRemoveMod(id); setConfirmRemove(null); }}>
                          Yes
                        </button>
                        <button className="btn btn-muted btn-sm" onClick={() => setConfirmRemove(null)}>
                          No
                        </button>
                      </>
                    ) : (
                      <>
                        <button className="btn btn-blurple btn-sm" onClick={() => setSelectedModID(id)}>
                          View Stats
                        </button>
                        <button className="btn btn-red btn-sm" onClick={() => setConfirmRemove(id)}>
                          Remove
                        </button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
