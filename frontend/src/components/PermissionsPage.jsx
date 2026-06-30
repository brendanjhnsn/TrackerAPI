import { useState, useEffect } from 'react';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

function MemberCell({ id, profiles }) {
  const avatarUrl = profiles[id]?.avatar_url;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      {avatarUrl && (
        <img src={avatarUrl} alt="" className="user-avatar" style={{ width: 24, height: 24 }} />
      )}
      <span>{profiles[id]?.username || id}</span>
    </div>
  );
}

const MANAGER_SECTIONS = [
  { key: 'moderators',       label: 'Moderators' },
  { key: 'management_panel', label: 'Management Panel' },
  { key: 'game_leads',       label: 'Game Leads' },
];

const MOD_SECTIONS = [
  { key: 'moderators', label: 'Moderators' },
  { key: 'game_leads', label: 'Game Leads' },
];

export default function PermissionsPage() {
  const [directors, setDirectors] = useState([]);
  const [managers, setManagers]   = useState([]);
  const [mods, setMods]           = useState([]);
  const [profiles, setProfiles]   = useState({});
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState(null);
  const [saving, setSaving]       = useState({}); // { "role:memberID:section": true }

  useEffect(() => {
    setLoading(true);
    fetch(`${BASE}/api/all-role-permissions`, { credentials: 'include' })
      .then(r => {
        if (!r.ok) throw new Error(`${r.status}`);
        return r.json();
      })
      .then(data => {
        setDirectors(Array.isArray(data.directors) ? data.directors : []);
        setManagers(Array.isArray(data.managers)   ? data.managers   : []);
        setMods(Array.isArray(data.mods)           ? data.mods       : []);

        const allIds = [
          ...(data.directors || []).map(d => d.member_id),
          ...(data.managers  || []).map(m => m.member_id),
          ...(data.mods      || []).map(m => m.member_id),
        ].filter(Boolean);

        if (allIds.length > 0) {
          fetch(`${BASE}/api/profiles?ids=${allIds.join(',')}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(list => {
              const map = {};
              for (const p of list) map[p.id] = p;
              setProfiles(map);
            })
            .catch(() => {});
        }
      })
      .catch(err => setError(`Failed to load permissions: ${err.message}`))
      .finally(() => setLoading(false));
  }, []);

  async function handleToggle(role, memberID, section, newValue) {
    const key = `${role}:${memberID}:${section}`;
    setSaving(prev => ({ ...prev, [key]: true }));

    const endpoint = role === 'manager' ? '/api/manager-permissions' : '/api/mod-permissions';
    const body = role === 'manager'
      ? { manager_id: memberID, section, enabled: newValue }
      : { member_id: memberID, section, enabled: newValue };

    // Optimistic update
    const setter = role === 'manager' ? setManagers : setMods;
    setter(prev => prev.map(m =>
      m.member_id === memberID
        ? { ...m, permissions: { ...m.permissions, [section]: newValue } }
        : m
    ));

    try {
      const res = await fetch(`${BASE}${endpoint}`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(`${res.status}`);
    } catch {
      // Revert on failure
      setter(prev => prev.map(m =>
        m.member_id === memberID
          ? { ...m, permissions: { ...m.permissions, [section]: !newValue } }
          : m
      ));
    } finally {
      setSaving(prev => { const next = { ...prev }; delete next[key]; return next; });
    }
  }

  if (loading) return <p className="loading-text">Loading permissions...</p>;
  if (error)   return <p style={{ color: 'var(--discord-danger, #f04747)', padding: '20px 0' }}>{error}</p>;

  const isEmpty = directors.length === 0 && managers.length === 0 && mods.length === 0;

  return (
    <section className="section">
      <h2 className="section-title">Permissions</h2>
      <p style={{ color: 'var(--discord-muted)', fontSize: 13, marginBottom: 20 }}>
        Toggle section access per member. Changes take effect immediately, including for members who haven't logged in yet.
      </p>

      {isEmpty && (
        <p className="loading-text">No members with Mod, Manager, or Director roles found.</p>
      )}

      {/* Directors */}
      {directors.length > 0 && (
        <div style={{ marginBottom: 28 }}>
          <h3 className="section-title" style={{ fontSize: 14, marginBottom: 10 }}>Directors</h3>
          <table className="loa-table">
            <thead>
              <tr>
                <th>Member</th>
                <th>Access</th>
              </tr>
            </thead>
            <tbody>
              {directors.map(d => (
                <tr key={d.member_id}>
                  <td><MemberCell id={d.member_id} profiles={profiles} /></td>
                  <td>
                    <span style={{
                      fontSize: 11, fontWeight: 600, padding: '2px 8px', borderRadius: 10,
                      background: 'var(--discord-blurple)', color: '#fff',
                      textTransform: 'uppercase', letterSpacing: 0.5,
                    }}>
                      Full Access
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Managers */}
      {managers.length > 0 && (
        <div style={{ marginBottom: 28 }}>
          <h3 className="section-title" style={{ fontSize: 14, marginBottom: 10 }}>Managers</h3>
          <table className="loa-table">
            <thead>
              <tr>
                <th>Member</th>
                {MANAGER_SECTIONS.map(s => (
                  <th key={s.key} style={{ textAlign: 'center' }}>{s.label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {managers.map(mgr => (
                <tr key={mgr.member_id}>
                  <td><MemberCell id={mgr.member_id} profiles={profiles} /></td>
                  {MANAGER_SECTIONS.map(s => {
                    const k = `manager:${mgr.member_id}:${s.key}`;
                    return (
                      <td key={s.key} style={{ textAlign: 'center' }}>
                        <input
                          type="checkbox"
                          checked={!!mgr.permissions?.[s.key]}
                          disabled={!!saving[k]}
                          onChange={e => handleToggle('manager', mgr.member_id, s.key, e.target.checked)}
                        />
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Mods */}
      {mods.length > 0 && (
        <div>
          <h3 className="section-title" style={{ fontSize: 14, marginBottom: 10 }}>Moderators</h3>
          <table className="loa-table">
            <thead>
              <tr>
                <th>Member</th>
                {MOD_SECTIONS.map(s => (
                  <th key={s.key} style={{ textAlign: 'center' }}>{s.label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {mods.map(mod => (
                <tr key={mod.member_id}>
                  <td><MemberCell id={mod.member_id} profiles={profiles} /></td>
                  {MOD_SECTIONS.map(s => {
                    const k = `mod:${mod.member_id}:${s.key}`;
                    return (
                      <td key={s.key} style={{ textAlign: 'center' }}>
                        <input
                          type="checkbox"
                          checked={!!mod.permissions?.[s.key]}
                          disabled={!!saving[k]}
                          onChange={e => handleToggle('mod', mod.member_id, s.key, e.target.checked)}
                        />
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
