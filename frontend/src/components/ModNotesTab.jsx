import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

function fmtDate(isoStr) {
  if (!isoStr) return '';
  return new Date(isoStr).toLocaleDateString();
}

function toInputDate(isoStr) {
  if (!isoStr) return '';
  return isoStr.split('T')[0];
}

export default function ModNotesTab({ isDirector, modIds, profiles, setProfiles }) {
  const { user } = useAuth();
  const [selectedModID, setSelectedModID] = useState('');

  const [notes, setNotes] = useState([]);
  const [notesLoading, setNotesLoading] = useState(false);

  const [training, setTraining] = useState(null);
  const [trainingLoading, setTrainingLoading] = useState(false);
  const [inTraining, setInTraining] = useState(false);
  const [trainingStart, setTrainingStart] = useState('');
  const [trainingEnd, setTrainingEnd] = useState('');
  const [trainingSaving, setTrainingSaving] = useState(false);
  const [trainingSaved, setTrainingSaved] = useState(false);

  const [newNote, setNewNote] = useState('');
  const [noteSubmitting, setNoteSubmitting] = useState(false);

  useEffect(() => {
    if (!selectedModID) return;

    setNotesLoading(true);
    fetch(`${BASE}/api/notes?mod_id=${selectedModID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(data => {
        setNotes(Array.isArray(data) ? data : []);
        const authorIds = [...new Set((Array.isArray(data) ? data : []).map(n => n.AuthorMemberID).filter(Boolean))];
        if (authorIds.length > 0) {
          fetch(`${BASE}/api/profiles?ids=${authorIds.join(',')}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(list => {
              setProfiles(prev => {
                const next = { ...prev };
                for (const p of list) next[p.id] = p;
                return next;
              });
            })
            .catch(() => {});
        }
      })
      .catch(() => setNotes([]))
      .finally(() => setNotesLoading(false));

    setTrainingLoading(true);
    fetch(`${BASE}/api/training?mod_id=${selectedModID}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        setTraining(data);
        setInTraining(data?.InTraining ?? false);
        setTrainingStart(toInputDate(data?.TrainingStart));
        setTrainingEnd(toInputDate(data?.TrainingEnd));
      })
      .catch(() => {})
      .finally(() => setTrainingLoading(false));
  }, [selectedModID]);

  async function handleSaveTraining(e) {
    e.preventDefault();
    setTrainingSaving(true);
    setTrainingSaved(false);
    try {
      const res = await fetch(`${BASE}/api/training`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mod_member_id: selectedModID,
          in_training: inTraining,
          training_start: trainingStart || null,
          training_end: trainingEnd || null,
        }),
      });
      if (res.ok) {
        setTraining(await res.json());
        setTrainingSaved(true);
        setTimeout(() => setTrainingSaved(false), 3000);
      }
    } catch (_) {}
    setTrainingSaving(false);
  }

  async function handleAddNote(e) {
    e.preventDefault();
    if (!newNote.trim()) return;
    setNoteSubmitting(true);
    try {
      const res = await fetch(`${BASE}/api/notes`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mod_member_id: selectedModID, content: newNote.trim() }),
      });
      if (res.ok) {
        const created = await res.json();
        setNotes(prev => [created, ...prev]);
        setNewNote('');
        if (user?.discord_user_id && !profiles[user.discord_user_id]) {
          fetch(`${BASE}/api/profiles?ids=${user.discord_user_id}`, { credentials: 'include' })
            .then(r => r.ok ? r.json() : [])
            .then(list => {
              setProfiles(prev => {
                const next = { ...prev };
                for (const p of list) next[p.id] = p;
                return next;
              });
            })
            .catch(() => {});
        }
      }
    } catch (_) {}
    setNoteSubmitting(false);
  }

  async function handleDeleteNote(id) {
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

  function displayName(id) {
    return profiles[id]?.username || id;
  }

  return (
    <div>
      <div className="form-group" style={{ marginBottom: 20, maxWidth: 340 }}>
        <label className="form-label">Select Mod</label>
        <select
          className="form-select"
          value={selectedModID}
          onChange={e => setSelectedModID(e.target.value)}
        >
          <option value="">— Choose a mod —</option>
          {modIds.map(id => (
            <option key={id} value={id}>{displayName(id)}</option>
          ))}
        </select>
      </div>

      {selectedModID && (
        <>
          {/* Training section */}
          <div style={{
            background: 'var(--discord-card)',
            borderRadius: 6,
            padding: 16,
            marginBottom: 20,
          }}>
            <p className="section-subtitle" style={{ margin: '0 0 12px' }}>Training Status</p>
            {trainingLoading ? (
              <p className="loading-text">Loading...</p>
            ) : (
              <form onSubmit={handleSaveTraining}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
                  <input
                    type="checkbox"
                    id="in-training"
                    checked={inTraining}
                    onChange={e => setInTraining(e.target.checked)}
                    style={{ width: 16, height: 16, cursor: 'pointer' }}
                  />
                  <label htmlFor="in-training" style={{ fontSize: 14, cursor: 'pointer' }}>
                    In Training
                  </label>
                </div>
                <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 14 }}>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label className="form-label">Training Start</label>
                    <input
                      type="date"
                      className="form-input"
                      style={{ padding: '6px 10px', fontSize: 13 }}
                      value={trainingStart}
                      onChange={e => setTrainingStart(e.target.value)}
                    />
                  </div>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label className="form-label">Training End</label>
                    <input
                      type="date"
                      className="form-input"
                      style={{ padding: '6px 10px', fontSize: 13 }}
                      value={trainingEnd}
                      onChange={e => setTrainingEnd(e.target.value)}
                    />
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                  <button type="submit" className="btn btn-blurple btn-sm" disabled={trainingSaving}>
                    {trainingSaving ? 'Saving...' : 'Save Training'}
                  </button>
                  {trainingSaved && (
                    <span style={{ fontSize: 13, color: 'var(--discord-green)' }}>Saved!</span>
                  )}
                </div>
              </form>
            )}
          </div>

          {/* Notes section */}
          <div style={{
            background: 'var(--discord-card)',
            borderRadius: 6,
            padding: 16,
          }}>
            <p className="section-subtitle" style={{ margin: '0 0 12px' }}>Notes for {displayName(selectedModID)}</p>

            <form onSubmit={handleAddNote} style={{ marginBottom: 20 }}>
              <div className="form-group" style={{ marginBottom: 10 }}>
                <label className="form-label">Add Note</label>
                <textarea
                  className="form-textarea"
                  placeholder="Write a note..."
                  value={newNote}
                  onChange={e => setNewNote(e.target.value)}
                  rows={3}
                />
              </div>
              <button type="submit" className="btn btn-blurple btn-sm" disabled={noteSubmitting || !newNote.trim()}>
                {noteSubmitting ? 'Adding...' : 'Add Note'}
              </button>
            </form>

            {notesLoading ? (
              <p className="loading-text">Loading notes...</p>
            ) : notes.length === 0 ? (
              <p className="muted">No notes yet.</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {notes.map(note => (
                  <div key={note.ID} style={{
                    background: 'var(--discord-darker)',
                    borderRadius: 4,
                    padding: '10px 12px',
                  }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 8 }}>
                      <p style={{ margin: 0, fontSize: 14, lineHeight: 1.5, whiteSpace: 'pre-wrap', flex: 1 }}>
                        {note.Content}
                      </p>
                      {isDirector && (
                        <button
                          className="btn btn-red btn-sm"
                          onClick={() => handleDeleteNote(note.ID)}
                          style={{ flexShrink: 0 }}
                        >
                          Delete
                        </button>
                      )}
                    </div>
                    <p style={{ margin: '6px 0 0', fontSize: 12, color: 'var(--discord-muted)' }}>
                      {displayName(note.AuthorMemberID)} · {fmtDate(note.CreatedAt)}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
