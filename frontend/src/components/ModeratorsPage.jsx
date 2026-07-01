import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { ViewSwitcher, BarChart, PieChart, CalendarHeatmap } from './StatsCharts';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const PRESETS = [
  { key: '7d',     label: '7 days' },
  { key: '30d',    label: '30 days' },
  { key: '90d',    label: '90 days' },
  { key: 'all',    label: 'All time' },
  { key: 'custom', label: 'Custom' },
];

function getQueryParams(range, start, end) {
  const now = new Date();
  const days = { '7d': 7, '30d': 30, '90d': 90 };
  if (days[range]) {
    const s = new Date(now);
    s.setDate(s.getDate() - days[range]);
    return { start_date: s.toISOString().split('T')[0], end_date: now.toISOString().split('T')[0] };
  }
  if (range === 'custom' && start && end) return { start_date: start, end_date: end };
  return {};
}

function buildQ(params) {
  const q = new URLSearchParams(params).toString();
  return q ? `?${q}` : '';
}

function fmtDate(iso) {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString(undefined, { timeZone: 'UTC' });
}

function toDateInput(iso) {
  if (!iso) return '';
  return iso.split('T')[0];
}

function DateRangePicker({ range, setRange, customStart, setCustomStart, customEnd, setCustomEnd }) {
  return (
    <>
      <div className="date-range-btns">
        {PRESETS.map(({ key, label }) => (
          <button key={key} className={`date-range-btn${range === key ? ' active' : ''}`} onClick={() => setRange(key)}>
            {label}
          </button>
        ))}
      </div>
      {range === 'custom' && (
        <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
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
    </>
  );
}

// ---- Detail view shown when a mod is clicked ----
function ModDetail({ modID, profiles, setProfiles, isDirector, onBack, onRemove }) {
  const [range, setRange]             = useState('30d');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd]     = useState('');
  const [stats, setStats]             = useState(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const [issuedActions, setIssuedActions]       = useState({ warning: 0, timeout: 0, kick: 0, ban: 0 });
  const [issuedLoading, setIssuedLoading]       = useState(false);
  const [chartView, setChartView]             = useState('num');
  const [calendarData, setCalendarData]       = useState([]);
  const [calendarLoading, setCalendarLoading] = useState(false);

  const [inTraining, setInTraining]     = useState(false);
  const [trainingStart, setTrainingStart] = useState('');
  const [trainingEnd, setTrainingEnd]   = useState('');
  const [trainingLoading, setTrainingLoading] = useState(false);
  const [trainingSaving, setTrainingSaving]   = useState(false);
  const [trainingSaved, setTrainingSaved]     = useState(false);

  const [notes, setNotes]             = useState([]);
  const [notesLoading, setNotesLoading] = useState(false);
  const [newNote, setNewNote]         = useState('');
  const [noteSaving, setNoteSaving]   = useState(false);

  const [confirmRemove, setConfirmRemove] = useState(false);

  const [actions, setActions]           = useState([]);
  const [actionsLoading, setActionsLoading] = useState(false);
  const [actionType, setActionType]     = useState('1_on_1');
  const [actionReason, setActionReason] = useState('');
  const [actionDate, setActionDate]     = useState(() => {
    const d = new Date();
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  });
  const [actionSaving, setActionSaving] = useState(false);

  // Fetch stats + issued action counts when range changes
  useEffect(() => {
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const params = getQueryParams(range, customStart, customEnd);
    const q = buildQ({ ...params, member_id: modID });

    setStatsLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages${q}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${q}`,  { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${q}`,      { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${q}`,    { credentials: 'include' }).then(r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice]) => {
        setStats({
          messages: Array.isArray(messages) ? messages.reduce((s, r) => s + (r.count    ?? 0), 0) : 0,
          tickets:  Array.isArray(tickets)  ? tickets.reduce( (s, r) => s + (r.tickets  ?? 0), 0) : 0,
          qa:       Array.isArray(qa)       ? qa.reduce(      (s, r) => s + (r.count    ?? 0), 0) : 0,
          voice:    Array.isArray(voice)    ? Math.round(voice.reduce((s, r) => s + (r.total_seconds ?? 0), 0) / 3600) : 0,
        });
      })
      .catch(() => {})
      .finally(() => setStatsLoading(false));

    setIssuedLoading(true);
    fetch(`${BASE}/api/mod-issued-actions${buildQ({ ...params, mod_id: modID })}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : {})
      .then(data => setIssuedActions({ warning: 0, timeout: 0, kick: 0, ban: 0, ...data }))
      .catch(() => {})
      .finally(() => setIssuedLoading(false));
  }, [modID, range, customStart, customEnd]);

  // Fetch calendar data when calendar view is active
  useEffect(() => {
    if (chartView !== 'cal') return;
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const controller = new AbortController();
    const params = getQueryParams(range, customStart, customEnd);
    const q = buildQ({ ...params, member_id: modID });
    setCalendarLoading(true);
    fetch(`${BASE}/api/daily-stats${q}`, { credentials: 'include', signal: controller.signal })
      .then(r => r.ok ? r.json() : [])
      .then(list => setCalendarData(Array.isArray(list) ? list : []))
      .catch(err => { if (err.name !== 'AbortError') setCalendarData([]); })
      .finally(() => setCalendarLoading(false));
    return () => controller.abort();
  }, [chartView, modID, range, customStart, customEnd]);

  // Fetch training + notes on mount
  useEffect(() => {
    setTrainingLoading(true);
    fetch(`${BASE}/api/training?mod_id=${modID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        if (data) {
          setInTraining(data.InTraining ?? false);
          setTrainingStart(toDateInput(data.TrainingStart));
          setTrainingEnd(toDateInput(data.TrainingEnd));
        }
      })
      .catch(() => {})
      .finally(() => setTrainingLoading(false));

    setActionsLoading(true);
    fetch(`${BASE}/api/mod-actions?mod_id=${modID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => setActions(Array.isArray(list) ? list : []))
      .catch(() => setActions([]))
      .finally(() => setActionsLoading(false));

    setNotesLoading(true);
    fetch(`${BASE}/api/notes?mod_id=${modID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => {
        const arr = Array.isArray(list) ? list : [];
        setNotes(arr);
        const authorIds = [...new Set(arr.map(n => n.AuthorMemberID).filter(Boolean))];
        if (authorIds.length > 0) {
          fetch(`${BASE}/api/profiles?ids=${authorIds.join(',')}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(ps => setProfiles(prev => {
              const next = { ...prev };
              for (const p of ps) next[p.id] = p;
              return next;
            }))
            .catch(() => {});
        }
      })
      .catch(() => setNotes([]))
      .finally(() => setNotesLoading(false));
  }, [modID]);

  async function saveTraining(e) {
    e.preventDefault();
    setTrainingSaving(true);
    setTrainingSaved(false);
    try {
      const res = await fetch(`${BASE}/api/training`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mod_member_id:  modID,
          in_training:    inTraining,
          training_start: trainingStart || null,
          training_end:   trainingEnd   || null,
        }),
      });
      if (res.ok) {
        setTrainingSaved(true);
        setTimeout(() => setTrainingSaved(false), 3000);
      }
    } catch (_) {}
    setTrainingSaving(false);
  }

  async function addAction(e) {
    e.preventDefault();
    setActionSaving(true);
    try {
      const res = await fetch(`${BASE}/api/mod-actions`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mod_member_id: modID,
          action_type:   actionType,
          reason:        actionReason.trim(),
          issued_at:     actionDate,
        }),
      });
      if (res.ok) {
        const created = await res.json();
        setActions(prev => [created, ...prev]);
        setActionReason('');
        setActionDate(new Date().toISOString().split('T')[0]);
      }
    } catch (_) {}
    setActionSaving(false);
  }

  async function deleteAction(id) {
    try {
      const res = await fetch(`${BASE}/api/mod-actions?id=${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.status === 204) setActions(prev => prev.filter(a => a.ID !== id));
    } catch (_) {}
  }

  async function addNote(e) {
    e.preventDefault();
    if (!newNote.trim()) return;
    setNoteSaving(true);
    try {
      const res = await fetch(`${BASE}/api/notes`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mod_member_id: modID, content: newNote.trim() }),
      });
      if (res.ok) {
        const created = await res.json();
        setNotes(prev => [created, ...prev]);
        setNewNote('');
      }
    } catch (_) {}
    setNoteSaving(false);
  }

  async function deleteNote(id) {
    try {
      const res = await fetch(`${BASE}/api/notes?id=${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.status === 204) {
        setNotes(prev => prev.filter(n => n.ID !== id));
      }
    } catch (_) {}
  }

  const p = profiles[modID];
  const name = p?.username || modID;

  return (
    <div>
      {/* Top bar */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 28, flexWrap: 'wrap' }}>
        <button className="btn btn-muted btn-sm" onClick={onBack}>← Back</button>
        {p?.avatar_url && <img src={p.avatar_url} alt="" className="user-avatar" style={{ width: 32, height: 32 }} />}
        <span style={{ fontSize: 17, fontWeight: 700 }}>{name}</span>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8, alignItems: 'center' }}>
          {confirmRemove ? (
            <>
              <span style={{ fontSize: 13, color: 'var(--discord-muted)' }}>Remove this mod?</span>
              <button className="btn btn-red btn-sm" onClick={() => { onRemove(modID); onBack(); }}>Confirm</button>
              <button className="btn btn-muted btn-sm" onClick={() => setConfirmRemove(false)}>Cancel</button>
            </>
          ) : (
            <button className="btn btn-red btn-sm" onClick={() => setConfirmRemove(true)}>Remove Mod</button>
          )}
        </div>
      </div>

      {/* Stats */}
      <section className="section" style={{ marginBottom: 20 }}>
        <h3 className="section-title" style={{ fontSize: 15 }}>Stats</h3>

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12, marginBottom: 4 }}>
          <DateRangePicker
            range={range} setRange={setRange}
            customStart={customStart} setCustomStart={setCustomStart}
            customEnd={customEnd} setCustomEnd={setCustomEnd}
          />
          <ViewSwitcher view={chartView} setView={setChartView} />
        </div>

        {statsLoading ? (
          <p className="loading-text">Loading...</p>
        ) : chartView === 'num' ? (
          <div className="stat-cards">
            <div className="stat-card">
              <div className="stat-card-value" style={{ color: '#7289da' }}>{(stats?.messages ?? 0).toLocaleString()}</div>
              <div className="stat-card-label">Messages</div>
            </div>
            <div className="stat-card">
              <div className="stat-card-value" style={{ color: '#43b581' }}>{(stats?.tickets ?? 0).toLocaleString()}</div>
              <div className="stat-card-label">Tickets</div>
            </div>
            <div className="stat-card">
              <div className="stat-card-value" style={{ color: '#faa61a' }}>{(stats?.qa ?? 0).toLocaleString()}</div>
              <div className="stat-card-label">Q&amp;A</div>
            </div>
            <div className="stat-card">
              <div className="stat-card-value" style={{ color: '#f04747' }}>{stats?.voice ?? 0}h</div>
              <div className="stat-card-label">Voice</div>
            </div>
          </div>
        ) : (
          <>
            {chartView === 'bar' && (
              <BarChart
                bars={[
                  { label: 'Messages',  value: stats?.messages          ?? 0, color: '#7289da' },
                  { label: 'Tickets',   value: stats?.tickets           ?? 0, color: '#43b581' },
                  { label: 'Q&A',       value: stats?.qa                ?? 0, color: '#faa61a' },
                  { label: 'Voice (h)', value: stats?.voice             ?? 0, color: '#f04747' },
                  { label: 'Warnings',  value: issuedActions.warning    ?? 0, color: '#faa61a' },
                  { label: 'Timeouts',  value: issuedActions.timeout    ?? 0, color: '#ff7043' },
                  { label: 'Kicks',     value: issuedActions.kick       ?? 0, color: '#ff9800' },
                  { label: 'Bans',      value: issuedActions.ban        ?? 0, color: '#ed4245' },
                ]}
              />
            )}
            {chartView === 'pie' && (
              <PieChart
                slices={[
                  { label: 'Messages',  value: stats?.messages          ?? 0, color: '#7289da' },
                  { label: 'Tickets',   value: stats?.tickets           ?? 0, color: '#43b581' },
                  { label: 'Q&A',       value: stats?.qa                ?? 0, color: '#faa61a' },
                  { label: 'Voice (h)', value: stats?.voice             ?? 0, color: '#f04747' },
                  { label: 'Warnings',  value: issuedActions.warning    ?? 0, color: '#faa61a' },
                  { label: 'Timeouts',  value: issuedActions.timeout    ?? 0, color: '#ff7043' },
                  { label: 'Kicks',     value: issuedActions.kick       ?? 0, color: '#ff9800' },
                  { label: 'Bans',      value: issuedActions.ban        ?? 0, color: '#ed4245' },
                ].filter(s => s.value > 0)}
              />
            )}
            {chartView === 'cal' && (
              calendarLoading
                ? <p className="loading-text">Loading calendar...</p>
                : <CalendarHeatmap dailyData={calendarData} baseColor="#7289da" />
            )}
          </>
        )}

        {chartView === 'num' && (
          <>
            <p className="section-subtitle" style={{ marginTop: 16 }}>Moderation Actions Issued</p>
            <div className="stat-cards">
              <div className="stat-card">
                <div className="stat-card-value" style={{ color: '#faa61a' }}>
                  {issuedLoading ? '—' : issuedActions.warning}
                </div>
                <div className="stat-card-label">Warnings</div>
              </div>
              <div className="stat-card">
                <div className="stat-card-value" style={{ color: '#ff7043' }}>
                  {issuedLoading ? '—' : issuedActions.timeout}
                </div>
                <div className="stat-card-label">Timeouts</div>
              </div>
              <div className="stat-card">
                <div className="stat-card-value" style={{ color: '#ff9800' }}>
                  {issuedLoading ? '—' : issuedActions.kick}
                </div>
                <div className="stat-card-label">Kicks</div>
              </div>
              <div className="stat-card">
                <div className="stat-card-value" style={{ color: '#f04747' }}>
                  {issuedLoading ? '—' : issuedActions.ban}
                </div>
                <div className="stat-card-label">Bans</div>
              </div>
            </div>
          </>
        )}
      </section>

      {/* Training */}
      <section className="section" style={{ marginBottom: 20 }}>
        <h3 className="section-title" style={{ fontSize: 15 }}>Training Status</h3>
        {trainingLoading ? (
          <p className="loading-text">Loading...</p>
        ) : (
          <form onSubmit={saveTraining}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
              <input
                type="checkbox"
                id={`in-training-${modID}`}
                checked={inTraining}
                onChange={e => setInTraining(e.target.checked)}
                style={{ width: 16, height: 16, cursor: 'pointer' }}
              />
              <label htmlFor={`in-training-${modID}`} style={{ fontSize: 14, cursor: 'pointer' }}>
                Currently in Training
              </label>
              {inTraining && (
                <span style={{
                  fontSize: 11, fontWeight: 600, padding: '2px 7px', borderRadius: 10,
                  background: 'var(--discord-green)', color: '#fff', textTransform: 'uppercase', letterSpacing: 0.5,
                }}>Active</span>
              )}
            </div>
            <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 14 }}>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">Start Date</label>
                <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13 }}
                  value={trainingStart} onChange={e => setTrainingStart(e.target.value)} />
              </div>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">End Date</label>
                <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13 }}
                  value={trainingEnd} onChange={e => setTrainingEnd(e.target.value)} />
              </div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <button type="submit" className="btn btn-blurple btn-sm" disabled={trainingSaving}>
                {trainingSaving ? 'Saving…' : 'Save Training'}
              </button>
              {trainingSaved && (
                <span style={{ fontSize: 13, color: 'var(--discord-green)' }}>Saved!</span>
              )}
            </div>
          </form>
        )}
      </section>

      {/* Actions */}
      <section className="section" style={{ marginBottom: 20 }}>
        <h3 className="section-title" style={{ fontSize: 15 }}>Actions</h3>

        <form onSubmit={addAction} style={{ marginBottom: 20 }}>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 10, alignItems: 'flex-end' }}>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Type</label>
              <select
                className="form-input"
                style={{ padding: '6px 10px', fontSize: 13 }}
                value={actionType}
                onChange={e => setActionType(e.target.value)}
              >
                <option value="1_on_1">1 on 1</option>
                <option value="review">Review</option>
                <option value="warning">Warning</option>
                <option value="performance_plan">Performance Plan</option>
              </select>
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Date</label>
              <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13 }}
                value={actionDate} onChange={e => setActionDate(e.target.value)} />
            </div>
          </div>
          <div className="form-group" style={{ marginBottom: 10 }}>
            <label className="form-label">Reason</label>
            <textarea className="form-textarea" placeholder="Reason for action…" rows={2}
              value={actionReason} onChange={e => setActionReason(e.target.value)} />
          </div>
          <button type="submit" className="btn btn-blurple btn-sm" disabled={actionSaving}>
            {actionSaving ? 'Adding…' : 'Log Action'}
          </button>
        </form>

        {actionsLoading ? (
          <p className="loading-text">Loading…</p>
        ) : actions.length === 0 ? (
          <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No actions logged.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {actions.map(action => {
              const badgeColor =
                action.ActionType === 'performance_plan' ? 'var(--discord-red)'
                : action.ActionType === 'warning'        ? 'var(--discord-yellow)'
                : action.ActionType === 'review'         ? '#ff7043'
                : '#7289da';
              return (
                <div key={action.ID} style={{
                  background: 'var(--discord-card)', borderRadius: 6,
                  padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)',
                  display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 10,
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: action.Reason ? 4 : 0 }}>
                      <span style={{
                        fontSize: 11, fontWeight: 700, padding: '2px 8px', borderRadius: 10,
                        background: badgeColor, color: '#fff', textTransform: 'uppercase', letterSpacing: 0.5,
                      }}>
                        {action.ActionType}
                      </span>
                      <span style={{ fontSize: 12, color: 'var(--discord-muted)' }}>
                        {fmtDate(action.IssuedAt)} · {profiles[action.AuthorMemberID]?.username || action.AuthorMemberID}
                      </span>
                    </div>
                    {action.Reason && (
                      <p style={{ margin: 0, fontSize: 13, lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>
                        {action.Reason}
                      </p>
                    )}
                  </div>
                  {isDirector && (
                    <button className="btn btn-red btn-sm" onClick={() => deleteAction(action.ID)}>
                      Delete
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </section>

      {/* Notes */}
      <section className="section">
        <h3 className="section-title" style={{ fontSize: 15 }}>Notes</h3>

        <form onSubmit={addNote} style={{ marginBottom: 20 }}>
          <div className="form-group" style={{ marginBottom: 10 }}>
            <label className="form-label">Add a Note</label>
            <textarea
              className="form-textarea"
              placeholder="Write a note about this mod…"
              rows={3}
              value={newNote}
              onChange={e => setNewNote(e.target.value)}
            />
          </div>
          <button type="submit" className="btn btn-blurple btn-sm" disabled={noteSaving || !newNote.trim()}>
            {noteSaving ? 'Adding…' : 'Add Note'}
          </button>
        </form>

        {notesLoading ? (
          <p className="loading-text">Loading notes…</p>
        ) : notes.length === 0 ? (
          <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No notes yet.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {notes.map(note => (
              <div key={note.ID} style={{
                background: 'var(--discord-card)', borderRadius: 6,
                padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)',
              }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 10 }}>
                  <p style={{ margin: 0, fontSize: 14, lineHeight: 1.5, whiteSpace: 'pre-wrap', flex: 1 }}>
                    {note.Content}
                  </p>
                  {isDirector && (
                    <button className="btn btn-red btn-sm" onClick={() => deleteNote(note.ID)}>
                      Delete
                    </button>
                  )}
                </div>
                <p style={{ margin: '6px 0 0', fontSize: 12, color: 'var(--discord-muted)' }}>
                  {profiles[note.AuthorMemberID]?.username || note.AuthorMemberID} · {fmtDate(note.CreatedAt)}
                </p>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

// ---- Mod list page ----
export default function ModeratorsPage() {
  const { user } = useAuth();
  const isDirector = user?.role === 'director';

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
  const [selectedModID, setSelectedModID] = useState(null);

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
          onBack={() => { setSelectedModID(null); setConfirmRemove(null); }}
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
                      onClick={() => setSelectedModID(row.member_id)}
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
                          onClick={() => setSelectedModID(row.member_id)}>
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
