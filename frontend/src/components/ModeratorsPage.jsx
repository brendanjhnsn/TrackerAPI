import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import ModDetail, { DateRangePicker, getQueryParams, buildQ } from './ModDetail';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

function modFromHash() {
  const parts = window.location.hash.slice(1).split('/');
  return parts[0] === 'moderators' && parts[1] ? parts[1] : null;
}

// ---- Mod list page ----
export default function ModeratorsPage() {
  const { user } = useAuth();
  const isDirector = user?.role === 'director';
  const isManager  = user?.role === 'manager';

  const [data, setData]             = useState([]);
  const [loading, setLoading]       = useState(true);
  const [profiles, setProfiles]     = useState({});
  const [removedIds, setRemovedIds] = useState(new Set());
  const [range, setRange]           = useState('30d');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd]   = useState('');
  const [sortCol, setSortCol]       = useState('messages');
  const [sortDir, setSortDir]       = useState('desc');
  const [confirmRemove, setConfirmRemove] = useState(null);
  const [selectedModID, setSelectedModIDState] = useState(modFromHash);

  useEffect(() => {
    const onHash = () => {
      const parts = window.location.hash.slice(1).split('/');
      if (parts[0] === 'moderators') setSelectedModIDState(parts[1] || null);
    };
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  function selectMod(id) {
    window.location.hash = `moderators/${id}`;
    setSelectedModIDState(id);
  }

  function clearMod() {
    window.location.hash = 'moderators';
    setSelectedModIDState(null);
  }

  // Load removed mods
  useEffect(() => {
    fetch(`${BASE}/api/removed-mods`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => Array.isArray(list) && setRemovedIds(new Set(list.map(r => r.MemberID))))
      .catch(() => {});
  }, []);

  // Fetch all metrics
  useEffect(() => {
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const q = buildQ(getQueryParams(range, customStart, customEnd));
    setLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${q}`).then( r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${q}`).then(     r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${q}`).then(   r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice]) => {
        const map = {};
        const zero = id => { if (!map[id]) map[id] = { member_id: id, messages: 0, tickets: 0, qa: 0, voice: 0 }; };
        for (const r of (Array.isArray(messages) ? messages : [])) { zero(r.member_id); map[r.member_id].messages += r.count    ?? 0; }
        for (const r of (Array.isArray(tickets)  ? tickets  : [])) { zero(r.member_id); map[r.member_id].tickets += r.tickets  ?? 0; }
        for (const r of (Array.isArray(qa)       ? qa       : [])) { zero(r.member_id); map[r.member_id].qa      += r.count    ?? 0; }
        for (const r of (Array.isArray(voice)    ? voice    : [])) { zero(r.member_id); map[r.member_id].voice   += Math.round((r.total_seconds ?? 0) / 3600); }
        setData(Object.values(map));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [range, customStart, customEnd]);

  // Fetch profiles
  useEffect(() => {
    if (data.length === 0) return;
    const ids = data.map(d => d.member_id).filter(Boolean);
    fetch(`${BASE}/api/profiles?ids=${ids.join(',')}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => setProfiles(prev => {
        const next = { ...prev };
        for (const p of list) next[p.id] = p;
        return next;
      }))
      .catch(() => {});
  }, [data]);

  async function handleRemoveMod(memberID) {
    try {
      const res = await fetch(`${BASE}/api/removed-mods`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ member_id: memberID }),
      });
      if (res.ok) setRemovedIds(prev => new Set([...prev, memberID]));
    } catch (_) {}
  }

  function handleSort(col) {
    if (sortCol === col) setSortDir(d => d === 'desc' ? 'asc' : 'desc');
    else { setSortCol(col); setSortDir('desc'); }
  }

  const activeMods = data.filter(d => !removedIds.has(d.member_id));
  const sorted = [...activeMods].sort((a, b) => {
    const diff = (b[sortCol] ?? 0) - (a[sortCol] ?? 0);
    return sortDir === 'desc' ? diff : -diff;
  });

  function arrow(col) {
    return sortCol === col ? (sortDir === 'desc' ? ' ▼' : ' ▲') : '';
  }

  // Show detail view when a mod is selected
  if (selectedModID) {
    return (
      <section className="section">
        <ModDetail
          modID={selectedModID}
          profiles={profiles}
          setProfiles={setProfiles}
          isDirector={isDirector}
          isManager={isManager}
          onBack={() => { clearMod(); setConfirmRemove(null); }}
          onRemove={handleRemoveMod}
        />
      </section>
    );
  }

  return (
    <section className="section">
      <h2 className="section-title">Moderators</h2>

      <DateRangePicker
        range={range} setRange={setRange}
        customStart={customStart} setCustomStart={setCustomStart}
        customEnd={customEnd} setCustomEnd={setCustomEnd}
      />

      {loading ? (
        <p className="loading-text">Loading…</p>
      ) : range === 'custom' && (!customStart || !customEnd) ? (
        <p className="loading-text">Select both dates to load data.</p>
      ) : sorted.length === 0 ? (
        <p className="loading-text">No mods found. Mods appear here once they have activity data.</p>
      ) : (
        <table className="loa-table">
          <thead>
            <tr>
              <th>Mod</th>
              <th style={{ cursor: 'pointer', userSelect: 'none' }} onClick={() => handleSort('messages')}>
                Messages{arrow('messages')}
              </th>
              <th style={{ cursor: 'pointer', userSelect: 'none' }} onClick={() => handleSort('tickets')}>
                Tickets{arrow('tickets')}
              </th>
              <th style={{ cursor: 'pointer', userSelect: 'none' }} onClick={() => handleSort('qa')}>
                Q&amp;A{arrow('qa')}
              </th>
              <th style={{ cursor: 'pointer', userSelect: 'none' }} onClick={() => handleSort('voice')}>
                Voice (h){arrow('voice')}
              </th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map(row => (
              <tr key={row.member_id}>
                <td>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {profiles[row.member_id]?.avatar_url && (
                      <img src={profiles[row.member_id].avatar_url} alt=""
                        className="user-avatar" style={{ width: 22, height: 22 }} />
                    )}
                    <button
                      style={{ background: 'none', border: 'none', color: 'var(--discord-text)',
                        cursor: 'pointer', padding: 0, fontSize: 14, fontWeight: 500 }}
                      onClick={() => selectMod(row.member_id)}
                    >
                      {profiles[row.member_id]?.username || row.member_id}
                    </button>
                  </div>
                </td>
                <td>{row.messages.toLocaleString()}</td>
                <td>{row.tickets.toLocaleString()}</td>
                <td>{row.qa.toLocaleString()}</td>
                <td>{row.voice}h</td>
                <td>
                  <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', alignItems: 'center' }}>
                    {confirmRemove === row.member_id ? (
                      <>
                        <span style={{ fontSize: 12, color: 'var(--discord-muted)' }}>Sure?</span>
                        <button className="btn btn-red btn-sm"
                          onClick={() => { handleRemoveMod(row.member_id); setConfirmRemove(null); }}>
                          Yes
                        </button>
                        <button className="btn btn-muted btn-sm" onClick={() => setConfirmRemove(null)}>
                          No
                        </button>
                      </>
                    ) : (
                      <>
                        <button className="btn btn-blurple btn-sm"
                          onClick={() => selectMod(row.member_id)}>
                          View
                        </button>
                        <button className="btn btn-red btn-sm"
                          onClick={() => setConfirmRemove(row.member_id)}>
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
    </section>
  );
}
