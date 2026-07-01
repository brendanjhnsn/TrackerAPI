import { useState, useEffect } from 'react';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const ACTION_TYPE_LABELS = {
  '1_on_1':           '1 On 1',
  'review':           'REVIEW',
  'warning':          'WARNING',
  'action_plan':      'ACTION PLAN',
  'performance_plan': 'PERFORMANCE PLAN',
};

function fmtDate(iso) {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString();
}

const TABS = [
  { key: 'notes',        label: 'Deleted Notes' },
  { key: 'actions',      label: 'Deleted Actions' },
  { key: 'removed_mods', label: 'Removed Mods' },
];

export default function AuditLogPage() {
  const [tab, setTab]           = useState('notes');
  const [search, setSearch]     = useState('');
  const [data, setData]         = useState([]);
  const [loading, setLoading]   = useState(false);
  const [profiles, setProfiles] = useState({});

  useEffect(() => { loadTab(tab); }, [tab]);

  async function loadTab(type) {
    setLoading(true);
    setData([]);
    setProfiles({});
    try {
      const res = await fetch(`${BASE}/api/audit-log?type=${type}`, { credentials: 'include' });
      if (!res.ok) { setLoading(false); return; }
      const rows = await res.json();
      setData(rows);

      const ids = new Set();
      rows.forEach(r => {
        [r.ModMemberID, r.MemberID, r.AuthorMemberID, r.DeletedBy, r.RemovedBy]
          .filter(Boolean).forEach(id => ids.add(id));
      });
      if (ids.size > 0) {
        const pRes = await fetch(`${BASE}/api/profiles?ids=${[...ids].join(',')}`, { credentials: 'include' });
        if (pRes.ok) {
          const pData = await pRes.json();
          const map = {};
          pData.forEach(p => { map[p.id] = p; });
          setProfiles(map);
        }
      }
    } catch (_) {}
    setLoading(false);
  }

  async function restoreMod(memberID) {
    try {
      const res = await fetch(`${BASE}/api/mod-restore`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ member_id: memberID }),
      });
      if (res.status === 204) {
        setData(prev => prev.filter(r => r.MemberID !== memberID));
      }
    } catch (_) {}
  }

  const name = id => profiles[id]?.username || id || '—';

  const filtered = data.filter(r => {
    if (!search.trim()) return true;
    const q = search.toLowerCase();
    const targetID   = (r.ModMemberID || r.MemberID || '').toLowerCase();
    const targetName = name(r.ModMemberID || r.MemberID).toLowerCase();
    return targetID.includes(q) || targetName.includes(q);
  });

  return (
    <section className="card">
      <h2 className="section-title">Audit Log</h2>

      <input
        className="form-input"
        style={{ marginBottom: 12, padding: '6px 10px', fontSize: 13, width: '100%', boxSizing: 'border-box' }}
        placeholder="Search by mod name or ID..."
        value={search}
        onChange={e => setSearch(e.target.value)}
      />

      <div style={{ display: 'flex', borderBottom: '2px solid var(--discord-bg)', marginBottom: 16 }}>
        {TABS.map(t => (
          <button
            key={t.key}
            onClick={() => { setTab(t.key); setSearch(''); }}
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              padding: '8px 16px', fontSize: 13, fontWeight: tab === t.key ? 600 : 400,
              color: tab === t.key ? '#fff' : 'var(--discord-muted)',
              borderBottom: tab === t.key ? '2px solid var(--discord-blurple)' : '2px solid transparent',
              marginBottom: -2,
            }}
          >
            {t.label}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="loading-text">Loading...</p>
      ) : filtered.length === 0 ? (
        <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No records found.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>

          {tab === 'notes' && filtered.map(note => (
            <div key={note.ID} style={{ background: 'var(--discord-card)', borderRadius: 6, padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)' }}>
              <div style={{ fontSize: 12, color: 'var(--discord-muted)', marginBottom: 4 }}>
                Deleted {fmtDate(note.DeletedAt)} · by {name(note.DeletedBy)} · Target: <strong style={{ color: 'var(--discord-text)' }}>{name(note.ModMemberID)}</strong>
              </div>
              <p style={{ margin: '4px 0 6px', fontSize: 13, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{note.Content}</p>
              <div style={{ fontSize: 11, color: 'var(--discord-muted)' }}>
                Originally by {name(note.AuthorMemberID)} · {fmtDate(note.CreatedAt)}
              </div>
            </div>
          ))}

          {tab === 'actions' && filtered.map(action => (
            <div key={action.ID} style={{ background: 'var(--discord-card)', borderRadius: 6, padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)' }}>
              <div style={{ fontSize: 12, color: 'var(--discord-muted)', marginBottom: 6 }}>
                Deleted {fmtDate(action.DeletedAt)} · by {name(action.DeletedBy)} · Target: <strong style={{ color: 'var(--discord-text)' }}>{name(action.ModMemberID)}</strong>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: action.Reason ? 6 : 0 }}>
                <span style={{ fontSize: 11, fontWeight: 700, padding: '2px 8px', borderRadius: 10, background: '#7289da', color: '#fff', letterSpacing: 0.5 }}>
                  {ACTION_TYPE_LABELS[action.ActionType] ?? action.ActionType.replace(/_/g, ' ').toUpperCase()}
                </span>
                <span style={{ fontSize: 12, color: 'var(--discord-muted)' }}>
                  {fmtDate(action.IssuedAt)} · by {name(action.AuthorMemberID)}
                </span>
              </div>
              {action.Reason && (
                <p style={{ margin: 0, fontSize: 13, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>{action.Reason}</p>
              )}
            </div>
          ))}

          {tab === 'removed_mods' && filtered.map(mod => (
            <div key={mod.ID} style={{ background: 'var(--discord-card)', borderRadius: 6, padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
              <div>
                <div style={{ fontSize: 13, color: 'var(--discord-text)' }}>
                  {name(mod.MemberID)}{' '}
                  <span style={{ fontSize: 11, color: 'var(--discord-muted)' }}>({mod.MemberID})</span>
                </div>
                <div style={{ fontSize: 12, color: 'var(--discord-muted)', marginTop: 2 }}>
                  Removed {fmtDate(mod.CreatedAt)} · by {name(mod.RemovedBy)}
                </div>
              </div>
              <button className="btn btn-blurple btn-sm" onClick={() => restoreMod(mod.MemberID)}>
                Restore
              </button>
            </div>
          ))}

        </div>
      )}
    </section>
  );
}
