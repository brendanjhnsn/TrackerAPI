import { useState, useEffect, useRef } from 'react';
import { ViewSwitcher, BarChart, PieChart, CalendarHeatmap } from './StatsCharts';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

export const ACTION_TYPE_LABELS = {
  '1_on_1':          '1 On 1',
  'review':          'REVIEW',
  'warning':         'WARNING',
  'action_plan':     'ACTION PLAN',
  'performance_plan':'PERFORMANCE PLAN',
};

const PRESETS = [
  { key: '7d',     label: '7 days' },
  { key: '30d',    label: '30 days' },
  { key: '90d',    label: '90 days' },
  { key: 'all',    label: 'All time' },
  { key: 'custom', label: 'Custom' },
];

export function getQueryParams(range, start, end) {
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

export function buildQ(params) {
  const q = new URLSearchParams(params).toString();
  return q ? `?${q}` : '';
}

function fmtDate(iso) {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString(undefined, { timeZone: 'UTC' });
}

function fmtTimestamp(iso) {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString();
}

function toDateInput(iso) {
  if (!iso) return '';
  return iso.split('T')[0];
}

export function DateRangePicker({ range, setRange, customStart, setCustomStart, customEnd, setCustomEnd }) {
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

function Modal({ open, onClose, title, children }) {
  useEffect(() => {
    if (!open) return;
    const onKey = e => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 1000,
        background: 'rgba(0,0,0,0.75)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '24px 16px',
      }}
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div style={{
        background: '#2c2f33',
        border: '1px solid rgba(255,255,255,0.08)',
        borderRadius: 10, padding: '24px 28px',
        width: '100%', maxWidth: 500,
        maxHeight: '90vh', overflowY: 'auto',
        boxShadow: '0 12px 48px rgba(0,0,0,0.6)',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <span style={{ fontSize: 16, fontWeight: 700, color: 'var(--discord-text)' }}>{title}</span>
          <button
            onClick={onClose}
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              color: 'var(--discord-muted)', fontSize: 22, lineHeight: 1, padding: '0 2px',
            }}
          >×</button>
        </div>
        {children}
      </div>
    </div>
  );
}

function AttachmentChip({ att, canDelete, onDelete }) {
  const isImage = att.MimeType.startsWith('image/');
  const url = `${BASE}/api/attachments/file?id=${att.ID}`;
  return (
    <div style={{
      display: 'inline-flex', alignItems: 'center', gap: 6,
      background: 'var(--discord-bg)', borderRadius: 6, padding: '4px 8px', fontSize: 12,
    }}>
      {isImage ? (
        <a href={url} target="_blank" rel="noopener noreferrer">
          <img src={url} alt={att.FileName}
            style={{ width: 48, height: 48, objectFit: 'cover', borderRadius: 4, display: 'block' }} />
        </a>
      ) : (
        <a href={url} download={att.FileName}
          style={{ color: 'var(--discord-blurple)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: 4 }}>
          <span>📎</span>
          <span style={{ maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {att.FileName}
          </span>
        </a>
      )}
      {canDelete && (
        <button
          onClick={() => onDelete(att.ID)}
          style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--discord-muted)', padding: 0, fontSize: 16, lineHeight: 1 }}
        >×</button>
      )}
    </div>
  );
}

export default function ModDetail({ modID, profiles, setProfiles, isDirector, isManager, readOnly = false, onBack, onRemove }) {
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
  const [noteModalOpen, setNoteModalOpen] = useState(false);

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
  const [actionModalOpen, setActionModalOpen] = useState(false);

  const [loas, setLoas]               = useState([]);
  const [loasLoading, setLoasLoading] = useState(false);
  const [loaStart, setLoaStart]       = useState('');
  const [loaEnd, setLoaEnd]           = useState('');
  const [loaReason, setLoaReason]     = useState('');
  const [loaSaving, setLoaSaving]     = useState(false);
  const [loaModalOpen, setLoaModalOpen] = useState(false);

  const noteFileInputRef   = useRef(null);
  const actionFileInputRef = useRef(null);
  const [noteFiles, setNoteFiles]                 = useState([]);
  const [actionFiles, setActionFiles]             = useState([]);
  const [noteAttachments, setNoteAttachments]     = useState({});
  const [actionAttachments, setActionAttachments] = useState({});

  useEffect(() => {
    if (!noteModalOpen) return;
    const onPaste = e => {
      const imgs = Array.from(e.clipboardData?.items ?? [])
        .filter(i => i.kind === 'file' && i.type.startsWith('image/'));
      if (imgs.length === 0) return;
      e.preventDefault();
      const files = imgs.map(item => {
        const f = item.getAsFile();
        if (!f) return null;
        const ext = f.type.split('/')[1] || 'png';
        return new File([f], `pasted-${Date.now()}.${ext}`, { type: f.type });
      }).filter(Boolean);
      setNoteFiles(prev => [...prev, ...files]);
    };
    document.addEventListener('paste', onPaste);
    return () => document.removeEventListener('paste', onPaste);
  }, [noteModalOpen]);

  useEffect(() => {
    if (!actionModalOpen) return;
    const onPaste = e => {
      const imgs = Array.from(e.clipboardData?.items ?? [])
        .filter(i => i.kind === 'file' && i.type.startsWith('image/'));
      if (imgs.length === 0) return;
      e.preventDefault();
      const files = imgs.map(item => {
        const f = item.getAsFile();
        if (!f) return null;
        const ext = f.type.split('/')[1] || 'png';
        return new File([f], `pasted-${Date.now()}.${ext}`, { type: f.type });
      }).filter(Boolean);
      setActionFiles(prev => [...prev, ...files]);
    };
    document.addEventListener('paste', onPaste);
    return () => document.removeEventListener('paste', onPaste);
  }, [actionModalOpen]);

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
      .then(list => {
        const arr = Array.isArray(list) ? list : [];
        setActions(arr);
        if (arr.length > 0) {
          const ids = arr.map(a => a.ID).join(',');
          fetch(`${BASE}/api/attachments?owner_type=action&owner_ids=${ids}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(atts => {
              const map = {};
              for (const a of (Array.isArray(atts) ? atts : [])) {
                if (!map[a.OwnerID]) map[a.OwnerID] = [];
                map[a.OwnerID].push(a);
              }
              setActionAttachments(map);
            })
            .catch(() => {});
        }
      })
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
        if (arr.length > 0) {
          const ids = arr.map(n => n.ID).join(',');
          fetch(`${BASE}/api/attachments?owner_type=note&owner_ids=${ids}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(atts => {
              const map = {};
              for (const a of (Array.isArray(atts) ? atts : [])) {
                if (!map[a.OwnerID]) map[a.OwnerID] = [];
                map[a.OwnerID].push(a);
              }
              setNoteAttachments(map);
            })
            .catch(() => {});
        }
      })
      .catch(() => setNotes([]))
      .finally(() => setNotesLoading(false));

    setLoasLoading(true);
    fetch(`${BASE}/api/loa?member_id=${modID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => setLoas(Array.isArray(list) ? list : []))
      .catch(() => setLoas([]))
      .finally(() => setLoasLoading(false));
  }, [modID]);

  function closeNoteModal() {
    setNoteModalOpen(false);
    setNewNote('');
    setNoteFiles([]);
    if (noteFileInputRef.current) noteFileInputRef.current.value = '';
  }

  function closeActionModal() {
    setActionModalOpen(false);
    setActionReason('');
    setActionFiles([]);
    if (actionFileInputRef.current) actionFileInputRef.current.value = '';
  }

  function closeLoaModal() {
    setLoaModalOpen(false);
    setLoaStart('');
    setLoaEnd('');
    setLoaReason('');
  }

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
        if (actionFiles.length > 0) {
          const newAtts = [];
          for (const file of actionFiles) {
            const fd = new FormData();
            fd.append('file', file);
            try {
              const uploadRes = await fetch(
                `${BASE}/api/attachments?owner_type=action&owner_id=${created.ID}`,
                { method: 'POST', credentials: 'include', body: fd }
              );
              if (uploadRes.ok) newAtts.push(await uploadRes.json());
            } catch (_) {}
          }
          if (newAtts.length > 0) {
            setActionAttachments(prev => ({
              ...prev,
              [created.ID]: [...(prev[created.ID] ?? []), ...newAtts],
            }));
          }
        }
        const _d = new Date();
        setActionDate(`${_d.getFullYear()}-${String(_d.getMonth() + 1).padStart(2, '0')}-${String(_d.getDate()).padStart(2, '0')}`);
        closeActionModal();
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
        if (noteFiles.length > 0) {
          const newAtts = [];
          for (const file of noteFiles) {
            const fd = new FormData();
            fd.append('file', file);
            try {
              const uploadRes = await fetch(
                `${BASE}/api/attachments?owner_type=note&owner_id=${created.ID}`,
                { method: 'POST', credentials: 'include', body: fd }
              );
              if (uploadRes.ok) newAtts.push(await uploadRes.json());
            } catch (_) {}
          }
          if (newAtts.length > 0) {
            setNoteAttachments(prev => ({
              ...prev,
              [created.ID]: [...(prev[created.ID] ?? []), ...newAtts],
            }));
          }
        }
        closeNoteModal();
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

  async function addLOA(e) {
    e.preventDefault();
    if (!loaStart || !loaEnd) return;
    setLoaSaving(true);
    try {
      const res = await fetch(`${BASE}/api/loa`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          member_id:  modID,
          reason:     loaReason.trim(),
          start_date: loaStart,
          end_date:   loaEnd,
        }),
      });
      if (res.ok) {
        const created = await res.json();
        setLoas(prev => [created, ...prev]);
        closeLoaModal();
      }
    } catch (_) {}
    setLoaSaving(false);
  }

  async function deleteLOA(id) {
    try {
      const res = await fetch(`${BASE}/api/loa?id=${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.status === 204) setLoas(prev => prev.filter(l => l.ID !== id));
    } catch (_) {}
  }

  async function deleteAttachment(ownerType, ownerID, id) {
    try {
      const res = await fetch(`${BASE}/api/attachments?id=${id}`, {
        method: 'DELETE', credentials: 'include',
      });
      if (res.status === 204) {
        const setter = ownerType === 'note' ? setNoteAttachments : setActionAttachments;
        setter(prev => ({
          ...prev,
          [ownerID]: (prev[ownerID] ?? []).filter(a => a.ID !== id),
        }));
      }
    } catch (_) {}
  }

  const p = profiles[modID];
  const name = p?.username || modID;

  const canManage = !readOnly && (isDirector || isManager);

  return (
    <div>
      {/* Top bar */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 28, flexWrap: 'wrap' }}>
        <button className="btn btn-muted btn-sm" onClick={onBack}>← Back</button>
        {p?.avatar_url && <img src={p.avatar_url} alt="" className="user-avatar" style={{ width: 32, height: 32 }} />}
        <span style={{ fontSize: 17, fontWeight: 700 }}>{name}</span>
        {readOnly && (
          <span style={{
            fontSize: 11, fontWeight: 600, padding: '2px 8px', borderRadius: 10,
            background: 'var(--discord-red)', color: '#fff', textTransform: 'uppercase', letterSpacing: 0.5,
          }}>Removed</span>
        )}
        {!readOnly && (
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
        )}
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
        ) : readOnly ? (
          <div style={{ fontSize: 14 }}>
            <p style={{ margin: '0 0 6px' }}>
              In Training: <strong>{inTraining ? 'Yes' : 'No'}</strong>
            </p>
            {trainingStart && <p style={{ margin: '0 0 4px', color: 'var(--discord-muted)', fontSize: 13 }}>Start: {trainingStart}</p>}
            {trainingEnd   && <p style={{ margin: 0, color: 'var(--discord-muted)', fontSize: 13 }}>End: {trainingEnd}</p>}
          </div>
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

      {/* Leave of Absence */}
      <section className="section" style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <h3 className="section-title" style={{ fontSize: 15, margin: 0 }}>Leave of Absence</h3>
          {canManage && (
            <button className="btn btn-blurple btn-sm" onClick={() => setLoaModalOpen(true)}>+ Add LOA</button>
          )}
        </div>
        {loasLoading ? (
          <p className="loading-text">Loading...</p>
        ) : loas.length === 0 ? (
          <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No LOA records.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {loas.map(loa => (
              <div key={loa.ID} style={{
                background: 'var(--discord-card)', borderRadius: 6,
                padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)',
                display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 10,
              }}>
                <div>
                  <div style={{ fontSize: 13, fontWeight: 600, marginBottom: loa.Reason ? 4 : 0 }}>
                    {fmtDate(loa.StartDate)} – {fmtDate(loa.EndDate)}
                  </div>
                  {loa.Reason && (
                    <p style={{ margin: 0, fontSize: 13, color: 'var(--discord-muted)', lineHeight: 1.4 }}>
                      {loa.Reason}
                    </p>
                  )}
                </div>
                {canManage && (
                  <button className="btn btn-red btn-sm" onClick={() => deleteLOA(loa.ID)}>Delete</button>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Actions */}
      <section className="section" style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <h3 className="section-title" style={{ fontSize: 15, margin: 0 }}>Actions</h3>
          {canManage && (
            <button className="btn btn-blurple btn-sm" onClick={() => setActionModalOpen(true)}>+ Log Action</button>
          )}
        </div>

        {actionsLoading ? (
          <p className="loading-text">Loading…</p>
        ) : actions.length === 0 ? (
          <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No actions logged.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {actions.map(action => {
              const badgeColor =
                action.ActionType === 'performance_plan' ? '#ff0000'
                : action.ActionType === 'action_plan'    ? 'var(--discord-red)'
                : action.ActionType === 'warning'        ? 'var(--discord-yellow)'
                : action.ActionType === 'review'         ? '#ff7043'
                : '#7289da';
              return (
                <div key={action.ID} style={{
                  background: 'var(--discord-card)', borderRadius: 6,
                  padding: '10px 14px', border: '1px solid rgba(255,255,255,0.05)',
                }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 10 }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: action.Reason ? 4 : 0 }}>
                        <span style={{
                          fontSize: 11, fontWeight: 700, padding: '2px 8px', borderRadius: 10,
                          background: badgeColor, color: '#fff', letterSpacing: 0.5,
                        }}>
                          {ACTION_TYPE_LABELS[action.ActionType] ?? action.ActionType.replace(/_/g, ' ').toUpperCase()}
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
                    {canManage && (
                      <button className="btn btn-red btn-sm" onClick={() => deleteAction(action.ID)}>
                        Delete
                      </button>
                    )}
                  </div>
                  {(actionAttachments[action.ID] ?? []).length > 0 && (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 8 }}>
                      {(actionAttachments[action.ID] ?? []).map(att => (
                        <AttachmentChip
                          key={att.ID}
                          att={att}
                          canDelete={canManage}
                          onDelete={id => deleteAttachment('action', action.ID, id)}
                        />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </section>

      {/* Notes */}
      <section className="section">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <h3 className="section-title" style={{ fontSize: 15, margin: 0 }}>Notes</h3>
          {canManage && (
            <button className="btn btn-blurple btn-sm" onClick={() => setNoteModalOpen(true)}>+ Add Note</button>
          )}
        </div>

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
                  {canManage && (
                    <button className="btn btn-red btn-sm" onClick={() => deleteNote(note.ID)}>
                      Delete
                    </button>
                  )}
                </div>
                <p style={{ margin: '6px 0 0', fontSize: 12, color: 'var(--discord-muted)' }}>
                  {profiles[note.AuthorMemberID]?.username || note.AuthorMemberID} · {fmtTimestamp(note.CreatedAt)}
                </p>
                {(noteAttachments[note.ID] ?? []).length > 0 && (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 8 }}>
                    {(noteAttachments[note.ID] ?? []).map(att => (
                      <AttachmentChip
                        key={att.ID}
                        att={att}
                        canDelete={canManage}
                        onDelete={id => deleteAttachment('note', note.ID, id)}
                      />
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      {/* ── Modals ── */}

      <Modal open={noteModalOpen} onClose={closeNoteModal} title="Add Note">
        <form onSubmit={addNote}>
          <div className="form-group" style={{ marginBottom: 14 }}>
            <label className="form-label">Note</label>
            <textarea
              className="form-textarea"
              placeholder="Write a note about this mod…"
              rows={4}
              value={newNote}
              onChange={e => setNewNote(e.target.value)}
              autoFocus
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginBottom: 8 }}>
            <input
              ref={noteFileInputRef}
              type="file"
              multiple
              accept="image/*,.pdf,.doc,.docx,.txt,.xlsx,.csv"
              style={{ display: 'none' }}
              onChange={e => setNoteFiles(Array.from(e.target.files))}
            />
            <button type="button" className="btn btn-muted btn-sm"
              onClick={() => noteFileInputRef.current?.click()}>
              📎 Attach files
            </button>
            <span style={{ fontSize: 12, color: 'var(--discord-muted)' }}>or paste an image (Ctrl+V)</span>
          </div>
          {noteFiles.length > 0 && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 14 }}>
              {noteFiles.map((f, i) => (
                <span key={i} style={{ fontSize: 12, background: 'var(--discord-bg)', padding: '2px 8px', borderRadius: 10, display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                  {f.name}
                  <button type="button"
                    onClick={() => setNoteFiles(prev => prev.filter((_, j) => j !== i))}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--discord-muted)', padding: 0 }}>×</button>
                </span>
              ))}
            </div>
          )}
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
            <button type="button" className="btn btn-muted btn-sm" onClick={closeNoteModal}>Cancel</button>
            <button type="submit" className="btn btn-blurple btn-sm" disabled={noteSaving || !newNote.trim()}>
              {noteSaving ? 'Adding…' : 'Add Note'}
            </button>
          </div>
        </form>
      </Modal>

      <Modal open={actionModalOpen} onClose={closeActionModal} title="Log Action">
        <form onSubmit={addAction}>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 14 }}>
            <div className="form-group" style={{ marginBottom: 0, flex: 1, minWidth: 140 }}>
              <label className="form-label">Type</label>
              <select
                className="form-input"
                style={{ padding: '6px 10px', fontSize: 13, width: '100%' }}
                value={actionType}
                onChange={e => setActionType(e.target.value)}
              >
                <option value="1_on_1">1 On 1</option>
                <option value="review">Review</option>
                <option value="warning">Warning</option>
                <option value="action_plan">Action Plan</option>
                <option value="performance_plan">Performance Plan</option>
              </select>
            </div>
            <div className="form-group" style={{ marginBottom: 0, flex: 1, minWidth: 140 }}>
              <label className="form-label">Date</label>
              <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13, width: '100%' }}
                value={actionDate} onChange={e => setActionDate(e.target.value)} />
            </div>
          </div>
          <div className="form-group" style={{ marginBottom: 14 }}>
            <label className="form-label">Reason</label>
            <textarea className="form-textarea" placeholder="Reason for action…" rows={3}
              value={actionReason} onChange={e => setActionReason(e.target.value)} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginBottom: 8 }}>
            <input
              ref={actionFileInputRef}
              type="file"
              multiple
              accept="image/*,.pdf,.doc,.docx,.txt,.xlsx,.csv"
              style={{ display: 'none' }}
              onChange={e => setActionFiles(Array.from(e.target.files))}
            />
            <button type="button" className="btn btn-muted btn-sm"
              onClick={() => actionFileInputRef.current?.click()}>
              📎 Attach files
            </button>
            <span style={{ fontSize: 12, color: 'var(--discord-muted)' }}>or paste an image (Ctrl+V)</span>
          </div>
          {actionFiles.length > 0 && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 14 }}>
              {actionFiles.map((f, i) => (
                <span key={i} style={{ fontSize: 12, background: 'var(--discord-bg)', padding: '2px 8px', borderRadius: 10, display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                  {f.name}
                  <button type="button"
                    onClick={() => setActionFiles(prev => prev.filter((_, j) => j !== i))}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--discord-muted)', padding: 0 }}>×</button>
                </span>
              ))}
            </div>
          )}
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
            <button type="button" className="btn btn-muted btn-sm" onClick={closeActionModal}>Cancel</button>
            <button type="submit" className="btn btn-blurple btn-sm" disabled={actionSaving}>
              {actionSaving ? 'Logging…' : 'Log Action'}
            </button>
          </div>
        </form>
      </Modal>

      <Modal open={loaModalOpen} onClose={closeLoaModal} title="Add Leave of Absence">
        <form onSubmit={addLOA}>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 14 }}>
            <div className="form-group" style={{ marginBottom: 0, flex: 1, minWidth: 140 }}>
              <label className="form-label">Start Date</label>
              <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13, width: '100%' }}
                value={loaStart} onChange={e => setLoaStart(e.target.value)} required />
            </div>
            <div className="form-group" style={{ marginBottom: 0, flex: 1, minWidth: 140 }}>
              <label className="form-label">End Date</label>
              <input type="date" className="form-input" style={{ padding: '6px 10px', fontSize: 13, width: '100%' }}
                value={loaEnd} onChange={e => setLoaEnd(e.target.value)} required />
            </div>
          </div>
          <div className="form-group" style={{ marginBottom: 14 }}>
            <label className="form-label">Reason (optional)</label>
            <textarea className="form-textarea" placeholder="Reason for leave…" rows={3}
              value={loaReason} onChange={e => setLoaReason(e.target.value)} />
          </div>
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
            <button type="button" className="btn btn-muted btn-sm" onClick={closeLoaModal}>Cancel</button>
            <button type="submit" className="btn btn-blurple btn-sm" disabled={loaSaving || !loaStart || !loaEnd}>
              {loaSaving ? 'Saving…' : 'Save LOA'}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
