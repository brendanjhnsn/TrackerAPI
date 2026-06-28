import { useState, useEffect, useCallback } from 'react';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

function computeStatus(startDate, endDate) {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const start = new Date(startDate);
  const end = new Date(endDate);
  if (start > today) return 'upcoming';
  if (end < today) return 'past';
  return 'active';
}

export default function ManagementPanel() {
  const [loas, setLoas] = useState([]);
  const [loaLoading, setLoaLoading] = useState(true);
  const [profiles, setProfiles] = useState({});

  const fetchLoas = useCallback(() => {
    setLoaLoading(true);
    fetch(`${BASE}/api/loa?all=true`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(data => {
        const list = Array.isArray(data) ? data : [];
        setLoas(list);

        const ids = [...new Set(list.map(l => l.MemberID).filter(Boolean))];
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
      .catch(() => setLoas([]))
      .finally(() => setLoaLoading(false));
  }, []);

  useEffect(() => { fetchLoas(); }, [fetchLoas]);

  async function handleDelete(id) {
    try {
      const res = await fetch(`${BASE}/api/loa?id=${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok || res.status === 204) fetchLoas();
    } catch {
      // ignore
    }
  }

  return (
    <section className="section">
      <h2 className="section-title">Management Panel</h2>

      <h3 className="section-subtitle">All LOA Requests</h3>
      {loaLoading ? (
        <p className="loading-text">Loading LOAs...</p>
      ) : loas.length === 0 ? (
        <p className="loading-text">No LOA requests found.</p>
      ) : (
        <table className="loa-table">
          <thead>
            <tr>
              <th>Member</th>
              <th>Start</th>
              <th>End</th>
              <th>Reason</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loas.map(loa => {
              const status = computeStatus(loa.StartDate, loa.EndDate);
              const p = profiles[loa.MemberID];
              return (
                <tr key={loa.ID}>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      {p?.avatar_url && (
                        <img src={p.avatar_url} alt="" className="user-avatar" style={{ width: 24, height: 24 }} />
                      )}
                      <span>{p?.username || loa.MemberID}</span>
                    </div>
                  </td>
                  <td>{loa.StartDate ? new Date(loa.StartDate).toLocaleDateString() : '—'}</td>
                  <td>{loa.EndDate ? new Date(loa.EndDate).toLocaleDateString() : '—'}</td>
                  <td>{loa.Reason || '—'}</td>
                  <td><span className={`badge badge-${status}`}>{status}</span></td>
                  <td>
                    <button
                      className="btn btn-red btn-sm"
                      onClick={() => handleDelete(loa.ID)}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </section>
  );
}
