import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';
const GUILD_ID = import.meta.env.VITE_GUILD_ID ?? '';

function formatDate(val) {
  if (!val) return '—';
  return new Date(val).toLocaleDateString();
}

export default function LoaForm() {
  const { user } = useAuth();
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [reason, setReason] = useState('');
  const [status, setStatus] = useState(null);
  const [errorMsg, setErrorMsg] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const [myLoas, setMyLoas] = useState([]);
  const [loasLoading, setLoasLoading] = useState(true);

  const fetchMyLoas = useCallback(() => {
    setLoasLoading(true);
    fetch(`${BASE}/api/loa?mine=true`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(data => setMyLoas(Array.isArray(data) ? data : []))
      .catch(() => setMyLoas([]))
      .finally(() => setLoasLoading(false));
  }, []);

  useEffect(() => {
    if (user) fetchMyLoas();
  }, [user, fetchMyLoas]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!user) return;
    setSubmitting(true);
    setStatus(null);
    try {
      const res = await fetch(`${BASE}/api/loa`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          guild_id: GUILD_ID,
          member_id: user.discord_user_id,
          start_date: startDate,
          end_date: endDate,
          reason,
        }),
      });
      if (res.ok) {
        setStatus('success');
        setStartDate('');
        setEndDate('');
        setReason('');
        fetchMyLoas();
      } else {
        const data = await res.json().catch(() => ({}));
        setErrorMsg(data.error ?? 'Submission failed');
        setStatus('error');
      }
    } catch {
      setErrorMsg('Network error');
      setStatus('error');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id) => {
    try {
      const res = await fetch(`${BASE}/api/loa?id=${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok || res.status === 204) {
        fetchMyLoas();
      }
    } catch {
      // ignore
    }
  };

  return (
    <section className="section">
      <h2 className="section-title">Submit LOA</h2>
      <form className="form" onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label" htmlFor="loa-start">Start Date</label>
          <input
            id="loa-start"
            type="date"
            className="form-input"
            value={startDate}
            onChange={e => setStartDate(e.target.value)}
            required
          />
        </div>
        <div className="form-group">
          <label className="form-label" htmlFor="loa-end">End Date</label>
          <input
            id="loa-end"
            type="date"
            className="form-input"
            value={endDate}
            onChange={e => setEndDate(e.target.value)}
            required
          />
        </div>
        <div className="form-group">
          <label className="form-label" htmlFor="loa-reason">Notes</label>
          <textarea
            id="loa-reason"
            className="form-textarea"
            value={reason}
            onChange={e => setReason(e.target.value)}
            placeholder="Add any notes here... (Optional)"
            rows={3}
          />
        </div>
        {status === 'success' && (
          <div className="banner banner-success">LOA submitted successfully!</div>
        )}
        {status === 'error' && (
          <div className="banner banner-error">{errorMsg}</div>
        )}
        <button type="submit" className="btn btn-blurple" disabled={submitting}>
          {submitting ? 'Submitting...' : 'Submit LOA'}
        </button>
      </form>

      <h3 className="section-subtitle">My LOAs</h3>
      {loasLoading ? (
        <p className="muted">Loading...</p>
      ) : myLoas.length === 0 ? (
        <p className="muted">No LOAs submitted yet.</p>
      ) : (
        <table className="loa-table">
          <thead>
            <tr>
              <th>Start</th>
              <th>End</th>
              <th>Reason</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {myLoas.map(loa => (
              <tr key={loa.ID}>
                <td>{formatDate(loa.StartDate)}</td>
                <td>{formatDate(loa.EndDate)}</td>
                <td>{loa.Reason || '—'}</td>
                <td>
                  <button
                    className="btn btn-red btn-sm"
                    onClick={() => handleDelete(loa.ID)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
