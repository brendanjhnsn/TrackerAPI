import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

function fmtDate(iso) {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString();
}

// Groups GameLeadDailyMessage rows by date, sums counts across channels
function groupByDate(messages) {
  const map = {};
  for (const m of messages) {
    const date = m.Date ? m.Date.split('T')[0] : '';
    if (!map[date]) map[date] = { date, total: 0, channels: {} };
    map[date].total += m.Count ?? 0;
    if (!map[date].channels[m.ChannelID]) map[date].channels[m.ChannelID] = 0;
    map[date].channels[m.ChannelID] += m.Count ?? 0;
  }
  return Object.values(map).sort((a, b) => b.date.localeCompare(a.date));
}

// ---- Detail view for a single GL ----
function GameLeadDetail({ glID, profiles, setProfiles, isDirector, onBack, availableChannels }) {
  const [assignments, setAssignments]         = useState([]);
  const [assignLoading, setAssignLoading]     = useState(true);
  const [selectedChannel, setSelectedChannel] = useState('');
  const [assignSaving, setAssignSaving]       = useState(false);

  const [messages, setMessages]               = useState([]);
  const [msgLoading, setMsgLoading]           = useState(true);
  const [expandedDate, setExpandedDate]       = useState(null);

  const [voice, setVoice]                     = useState([]);
  const [voiceLoading, setVoiceLoading]       = useState(true);

  const [notes, setNotes]                     = useState([]);
  const [notesLoading, setNotesLoading]       = useState(true);
  const [newNote, setNewNote]                 = useState('');
  const [noteSaving, setNoteSaving]           = useState(false);

  useEffect(() => {
    setAssignLoading(true);
    fetch(`${BASE}/api/game-lead-assignments?user_id=${glID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => setAssignments(Array.isArray(list) ? list : []))
      .catch(() => setAssignments([]))
      .finally(() => setAssignLoading(false));

    setMsgLoading(true);
    fetch(`${BASE}/api/game-lead-messages?user_id=${glID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => setMessages(Array.isArray(list) ? list : []))
      .catch(() => setMessages([]))
      .finally(() => setMsgLoading(false));

    setVoiceLoading(true);
    fetch(`${BASE}/api/game-lead-voice?user_id=${glID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => setVoice(Array.isArray(list) ? list : []))
      .catch(() => setVoice([]))
      .finally(() => setVoiceLoading(false));

    setNotesLoading(true);
    fetch(`${BASE}/api/game-lead-notes?user_id=${glID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => {
        const arr = Array.isArray(list) ? list : [];
        setNotes(arr);
        const authorIds = [...new Set(arr.map(n => n.AuthorID).filter(Boolean))];
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
  }, [glID]);

  async function addAssignment() {
    if (!selectedChannel) return;
    setAssignSaving(true);
    try {
      const res = await fetch(`${BASE}/api/game-lead-assignments`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: glID, channel_id: selectedChannel }),
      });
      if (res.ok) {
        const created = await res.json();
        setAssignments(prev => [...prev, created]);
        setSelectedChannel('');
      }
    } catch (_) {}
    setAssignSaving(false);
  }

  async function removeAssignment(channelID) {
    try {
      const res = await fetch(
        `${BASE}/api/game-lead-assignments?user_id=${glID}&channel_id=${channelID}`,
        { method: 'DELETE', credentials: 'include' }
      );
      if (res.status === 204) {
        setAssignments(prev => prev.filter(a => a.ChannelID !== channelID));
      }
    } catch (_) {}
  }

  async function addNote(e) {
    e.preventDefault();
    if (!newNote.trim()) return;
    setNoteSaving(true);
    try {
      const res = await fetch(`${BASE}/api/game-lead-notes`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: glID, text: newNote.trim() }),
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
      const res = await fetch(`${BASE}/api/game-lead-notes?id=${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.status === 204) setNotes(prev => prev.filter(n => n.ID !== id));
    } catch (_) {}
  }

  const p = profiles[glID];
  const name = p?.username || glID;
  const assignedIDs = new Set(assignments.map(a => a.ChannelID));
  const unassignedChannels = availableChannels.filter(c => !assignedIDs.has(c.id));
  const dailyMessages = groupByDate(messages);

  return (
    <div>
      {/* Top bar */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 28, flexWrap: 'wrap' }}>
        <button className="btn btn-muted btn-sm" onClick={onBack}>← Back</button>
        {p?.avatar_url && <img src={p.avatar_url} alt="" className="user-avatar" style={{ width: 32, height: 32 }} />}
        <span style={{ fontSize: 17, fontWeight: 700 }}>{name}</span>
      </div>

      {/* Channel Assignments */}
      <section className="section" style={{ marginBottom: 20 }}>
        <h3 className="section-title" style={{ fontSize: 15 }}>Assigned Channels</h3>

        {assignLoading ? (
          <p className="loading-text">Loading…</p>
        ) : (
          <>
            {assignments.length === 0 ? (
              <p style={{ color: 'var(--discord-muted)', fontSize: 14, marginBottom: 12 }}>No channels assigned.</p>
            ) : (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 12 }}>
                {assignments.map(a => {
                  const ch = availableChannels.find(c => c.id === a.ChannelID);
                  return (
                    <span key={a.ChannelID} style={{
                      display: 'inline-flex', alignItems: 'center', gap: 6,
                      background: 'var(--discord-card)', borderRadius: 4,
                      padding: '4px 10px', fontSize: 13, border: '1px solid rgba(255,255,255,0.08)',
                    }}>
                      #{ch?.name || a.ChannelID}
                      <button
                        onClick={() => removeAssignment(a.ChannelID)}
                        style={{ background: 'none', border: 'none', color: 'var(--discord-muted)',
                          cursor: 'pointer', padding: 0, fontSize: 14, lineHeight: 1 }}
                        title="Remove"
                      >×</button>
                    </span>
                  );
                })}
              </div>
            )}

            {unassignedChannels.length > 0 && (
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <select
                  className="form-input"
                  style={{ padding: '6px 10px', fontSize: 13 }}
                  value={selectedChannel}
                  onChange={e => setSelectedChannel(e.target.value)}
                >
                  <option value="">— Select channel —</option>
                  {unassignedChannels.map(c => (
                    <option key={c.id} value={c.id}>#{c.name}</option>
                  ))}
                </select>
                <button
                  className="btn btn-blurple btn-sm"
                  onClick={addAssignment}
                  disabled={!selectedChannel || assignSaving}
                >
                  {assignSaving ? 'Adding…' : 'Assign'}
                </button>
              </div>
            )}
          </>
        )}
      </section>

      {/* Message Activity */}
      <section className="section" style={{ marginBottom: 20 }}>
        <h3 className="section-title" style={{ fontSize: 15 }}>Message Activity</h3>
        {msgLoading ? (
          <p className="loading-text">Loading…</p>
        ) : dailyMessages.length === 0 ? (
          <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No message activity recorded.</p>
        ) : (
          <table className="loa-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Total Messages</th>
                <th>Channels</th>
              </tr>
            </thead>
            <tbody>
              {dailyMessages.map(row => (
                <React.Fragment key={row.date}>
                  <tr>
                    <td>{row.date}</td>
                    <td>{row.total.toLocaleString()}</td>
                    <td>
                      {Object.keys(row.channels).length > 1 ? (
                        <button
                          style={{ background: 'none', border: 'none', color: 'var(--discord-blurple)',
                            cursor: 'pointer', fontSize: 13, padding: 0 }}
                          onClick={() => setExpandedDate(expandedDate === row.date ? null : row.date)}
                        >
                          {Object.keys(row.channels).length} channels {expandedDate === row.date ? '▲' : '▼'}
                        </button>
                      ) : (
                        <span style={{ fontSize: 13, color: 'var(--discord-muted)' }}>
                          {(() => {
                            const chID = Object.keys(row.channels)[0];
                            const ch = availableChannels.find(c => c.id === chID);
                            return ch ? `#${ch.name}` : chID;
                          })()}
                        </span>
                      )}
                    </td>
                  </tr>
                  {expandedDate === row.date && (
                    Object.entries(row.channels).map(([chID, count]) => {
                      const ch = availableChannels.find(c => c.id === chID);
                      return (
                        <tr key={`${row.date}-${chID}`} style={{ background: 'rgba(255,255,255,0.02)' }}>
                          <td style={{ paddingLeft: 24, color: 'var(--discord-muted)', fontSize: 13 }}>
                            #{ch?.name || chID}
                          </td>
                          <td style={{ fontSize: 13 }}>{count.toLocaleString()}</td>
                          <td />
                        </tr>
                      );
                    })
                  )}
                </React.Fragment>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* Voice Time */}
      <section className="section" style={{ marginBottom: 20 }}>
        <h3 className="section-title" style={{ fontSize: 15 }}>Voice Time</h3>
        {voiceLoading ? (
          <p className="loading-text">Loading…</p>
        ) : voice.length === 0 ? (
          <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No voice activity recorded.</p>
        ) : (
          <table className="loa-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Hours</th>
                <th>Minutes</th>
              </tr>
            </thead>
            <tbody>
              {voice.map(row => (
                <tr key={row.date}>
                  <td>{row.date}</td>
                  <td>{row.hours}h</td>
                  <td>{row.minutes}m</td>
                </tr>
              ))}
            </tbody>
          </table>
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
              placeholder="Notes about this Game Lead…"
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
                    {note.Text}
                  </p>
                  {isDirector && (
                    <button className="btn btn-red btn-sm" onClick={() => deleteNote(note.ID)}>
                      Delete
                    </button>
                  )}
                </div>
                <p style={{ margin: '6px 0 0', fontSize: 12, color: 'var(--discord-muted)' }}>
                  {profiles[note.AuthorID]?.username || note.AuthorID} · {fmtDate(note.CreatedAt)}
                </p>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

// ---- List view ----
export default function GameLeadsPage() {
  const { user } = useAuth();
  const isDirector = user?.role === 'director';

  const [glIDs, setGlIDs]               = useState([]);
  const [loading, setLoading]           = useState(true);
  const [profiles, setProfiles]         = useState({});
  const [availableChannels, setAvailableChannels] = useState([]);
  const [assignments, setAssignments]   = useState({});  // { [userID]: [ChannelID, ...] }
  const [selectedGL, setSelectedGL]     = useState(null);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      fetch(`${BASE}/api/game-leads`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/game-leads/channels`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
    ])
      .then(([ids, channels]) => {
        const idList = Array.isArray(ids) ? ids : [];
        setGlIDs(idList);
        setAvailableChannels(Array.isArray(channels) ? channels : []);

        if (idList.length > 0) {
          fetch(`${BASE}/api/profiles?ids=${idList.join(',')}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(ps => {
              const map = {};
              for (const p of (Array.isArray(ps) ? ps : [])) map[p.id] = p;
              setProfiles(map);
            })
            .catch(() => {});

          Promise.all(
            idList.map(id =>
              fetch(`${BASE}/api/game-lead-assignments?user_id=${id}`, { credentials: 'include' })
                .then(r => r.ok ? r.json() : [])
                .then(list => ({ id, list: Array.isArray(list) ? list : [] }))
                .catch(() => ({ id, list: [] }))
            )
          ).then(results => {
            const map = {};
            for (const { id, list } of results) map[id] = list.map(a => a.ChannelID);
            setAssignments(map);
          });
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (selectedGL) {
    return (
      <GameLeadDetail
        glID={selectedGL}
        profiles={profiles}
        setProfiles={setProfiles}
        isDirector={isDirector}
        onBack={() => setSelectedGL(null)}
        availableChannels={availableChannels}
      />
    );
  }

  return (
    <section className="section">
      <h2 className="section-title">Game Leads</h2>

      {loading ? (
        <p className="loading-text">Loading…</p>
      ) : glIDs.length === 0 ? (
        <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No Game Leads found.</p>
      ) : (
        <table className="loa-table">
          <thead>
            <tr>
              <th>Game Lead</th>
              <th>Assigned Channels</th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {glIDs.map(id => {
              const p = profiles[id];
              const chIDs = assignments[id] ?? [];
              const chNames = chIDs.map(cid => {
                const ch = availableChannels.find(c => c.id === cid);
                return ch ? `#${ch.name}` : cid;
              });
              return (
                <tr key={id}>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      {p?.avatar_url && (
                        <img src={p.avatar_url} alt="" className="user-avatar" style={{ width: 22, height: 22 }} />
                      )}
                      <span style={{ fontSize: 14, fontWeight: 500 }}>
                        {p?.username || id}
                      </span>
                    </div>
                  </td>
                  <td style={{ fontSize: 13, color: chNames.length ? 'inherit' : 'var(--discord-muted)' }}>
                    {chNames.length ? chNames.join(', ') : 'None'}
                  </td>
                  <td>
                    <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                      <button
                        className="btn btn-blurple btn-sm"
                        onClick={() => setSelectedGL(id)}
                      >
                        View
                      </button>
                    </div>
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
