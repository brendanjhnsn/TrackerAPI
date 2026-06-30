import { useState, useEffect } from 'react';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const SECTIONS = [
  { key: 'moderators', label: 'Moderators' },
  { key: 'management_panel', label: 'Management Panel' },
  { key: 'game_leads', label: 'Game Leads' },
];

export default function PermissionsPage() {
  const [managers, setManagers] = useState([]);
  const [profiles, setProfiles] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState({}); // { "managerID:section": true } while saving

  useEffect(() => {
    setLoading(true);
    fetch(`${BASE}/api/manager-permissions`, { credentials: 'include' })
      .then(r => {
        if (!r.ok) throw new Error(`${r.status}`);
        return r.json();
      })
      .then(data => {
        const list = Array.isArray(data) ? data : [];
        setManagers(list);
        const ids = list.map(m => m.manager_id).filter(Boolean);
        if (ids.length > 0) {
          fetch(`${BASE}/api/profiles?ids=${ids.join(',')}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(profileList => {
              const map = {};
              for (const p of profileList) map[p.id] = p;
              setProfiles(map);
            })
            .catch(() => {});
        }
      })
      .catch(err => setError(`Failed to load managers: ${err.message}`))
      .finally(() => setLoading(false));
  }, []);

  async function handleToggle(managerID, section, newValue) {
    const key = `${managerID}:${section}`;
    setSaving(prev => ({ ...prev, [key]: true }));

    // optimistic update
    setManagers(prev =>
      prev.map(m =>
        m.manager_id === managerID
          ? { ...m, permissions: { ...m.permissions, [section]: newValue } }
          : m
      )
    );

    try {
      const res = await fetch(`${BASE}/api/manager-permissions`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ manager_id: managerID, section, enabled: newValue }),
      });
      if (!res.ok) throw new Error(`${res.status}`);
    } catch {
      // revert optimistic update on failure
      setManagers(prev =>
        prev.map(m =>
          m.manager_id === managerID
            ? { ...m, permissions: { ...m.permissions, [section]: !newValue } }
            : m
        )
      );
    } finally {
      setSaving(prev => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
    }
  }

  if (loading) return <p className="loading-text">Loading managers...</p>;
  if (error) return <p className="loading-text">{error}</p>;
  if (managers.length === 0) return (
    <section className="section">
      <h2 className="section-title">Manager Permissions</h2>
      <p className="loading-text">No managers found in Discord.</p>
    </section>
  );

  return (
    <section className="section">
      <h2 className="section-title">Manager Permissions</h2>
      <p className="loading-text" style={{ marginBottom: 16 }}>
        Toggle which sections each Manager can see. Directors always have full access. Changes take effect immediately.
      </p>
      <table className="loa-table">
        <thead>
          <tr>
            <th>Manager</th>
            {SECTIONS.map(s => <th key={s.key}>{s.label}</th>)}
          </tr>
        </thead>
        <tbody>
          {managers.map(mgr => {
            const p = profiles[mgr.manager_id];
            return (
              <tr key={mgr.manager_id}>
                <td>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {p?.avatar_url && (
                      <img src={p.avatar_url} alt="" className="user-avatar" style={{ width: 24, height: 24 }} />
                    )}
                    <span>{p?.username || mgr.manager_id}</span>
                  </div>
                </td>
                {SECTIONS.map(s => {
                  const key = `${mgr.manager_id}:${s.key}`;
                  return (
                    <td key={s.key} style={{ textAlign: 'center' }}>
                      <input
                        type="checkbox"
                        checked={!!mgr.permissions?.[s.key]}
                        disabled={!!saving[key]}
                        onChange={e => handleToggle(mgr.manager_id, s.key, e.target.checked)}
                      />
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}
