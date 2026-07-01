import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import ModNotesTab from './ModNotesTab';
import ModeratorsTab from './ModeratorsTab';
import { ViewSwitcher, BarChart, PieChart, CalendarHeatmap } from './StatsCharts';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const TABS = ['Moderators', 'Messages', 'Tickets', 'Q&A', 'Voice Hours', 'Warnings', 'Timeouts', 'Kicks', 'Bans', 'Mod Notes'];
const METRIC_KEY = {
  Messages: 'messages', Tickets: 'tickets', 'Q&A': 'qa', 'Voice Hours': 'voice',
  Warnings: 'warning', Timeouts: 'timeout', Kicks: 'kick', Bans: 'ban',
};
const METRIC_COLOR = {
  messages: '#7289da', tickets: '#43b581', qa: '#faa61a', voice: '#f04747',
  warning: '#faa61a', timeout: '#ff7043', kick: '#ff9800', ban: '#ed4245',
};

const PIE_COLORS = ['#7289da','#43b581','#faa61a','#f04747','#b9bbbe','#ed4245','#5865f2','#eb459e'];

function getQueryParams(range, customStart, customEnd) {
  const now = new Date();
  if (range === '7d') {
    const start = new Date(now);
    start.setDate(start.getDate() - 7);
    return { start_date: start.toISOString().split('T')[0], end_date: now.toISOString().split('T')[0] };
  }
  if (range === '30d') {
    const start = new Date(now);
    start.setDate(start.getDate() - 30);
    return { start_date: start.toISOString().split('T')[0], end_date: now.toISOString().split('T')[0] };
  }
  if (range === '90d') {
    const start = new Date(now);
    start.setDate(start.getDate() - 90);
    return { start_date: start.toISOString().split('T')[0], end_date: now.toISOString().split('T')[0] };
  }
  if (range === 'custom' && customStart && customEnd) {
    return { start_date: customStart, end_date: customEnd };
  }
  return {};
}

function buildQuery(params) {
  const q = new URLSearchParams(params).toString();
  return q ? `?${q}` : '';
}

const EMPTY_ROW = () => ({ messages: 0, tickets: 0, qa: 0, voice: 0, warning: 0, timeout: 0, kick: 0, ban: 0 });

function buildData(messages, tickets, qa, voice, actions) {
  const data = {};
  for (const row of messages) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, ...EMPTY_ROW() };
    data[row.member_id].messages += row.count ?? 0;
  }
  for (const row of tickets) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, ...EMPTY_ROW() };
    data[row.member_id].tickets += row.tickets ?? 0;
  }
  for (const row of qa) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, ...EMPTY_ROW() };
    data[row.member_id].qa += row.count ?? 0;
  }
  for (const row of voice) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, ...EMPTY_ROW() };
    data[row.member_id].voice += Math.round((row.total_seconds ?? 0) / 3600);
  }
  for (const row of actions) {
    if (!data[row.member_id]) data[row.member_id] = { member_id: row.member_id, ...EMPTY_ROW() };
    data[row.member_id].warning += row.warning ?? 0;
    data[row.member_id].timeout += row.timeout ?? 0;
    data[row.member_id].kick    += row.kick    ?? 0;
    data[row.member_id].ban     += row.ban     ?? 0;
  }
  return Object.values(data);
}

export default function ManagementView() {
  const { user } = useAuth();
  const isDirector = user?.role === 'director';

  const [activeTab, setActiveTab] = useState('Moderators');
  const [range, setRange] = useState('30d');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [profiles, setProfiles] = useState({});
  const [sortDir, setSortDir] = useState('desc');
  const [removedIds, setRemovedIds] = useState(new Set());
  const [chartView, setChartView]             = useState('bar');
  const [calendarData, setCalendarData]       = useState([]);
  const [calendarLoading, setCalendarLoading] = useState(false);

  const presets = [
    { key: '7d', label: 'Last 7 days' },
    { key: '30d', label: 'Last 30 days' },
    { key: '90d', label: 'Last 90 days' },
    { key: 'all', label: 'All time' },
    { key: 'custom', label: 'Custom' },
  ];

  // Load removed mods once on mount
  useEffect(() => {
    fetch(`${BASE}/api/removed-mods`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => {
        if (Array.isArray(list)) {
          setRemovedIds(new Set(list.map(r => r.MemberID)));
        }
      })
      .catch(() => {});
  }, []);

  // Fetch all metric data
  useEffect(() => {
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const q = buildQuery(getQueryParams(range, customStart, customEnd));
    setLoading(true);
    Promise.all([
      fetch(`${BASE}/api/messages${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/tickets${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/qfs${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/voice${q}`).then(r => r.ok ? r.json() : []),
      fetch(`${BASE}/api/mod-issued-actions${q}`, { credentials: 'include' }).then(r => r.ok ? r.json() : []),
    ])
      .then(([messages, tickets, qa, voice, actions]) => {
        setData(buildData(
          Array.isArray(messages) ? messages : [],
          Array.isArray(tickets)  ? tickets  : [],
          Array.isArray(qa)       ? qa       : [],
          Array.isArray(voice)    ? voice    : [],
          Array.isArray(actions)  ? actions  : [],
        ));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [range, customStart, customEnd]);

  // Fetch profiles for all mods
  useEffect(() => {
    if (data.length === 0) return;
    const ids = data.map(d => d.member_id).filter(Boolean);
    fetch(`${BASE}/api/profiles?ids=${ids.join(',')}`, { credentials: 'include' })
      .then(r => r.ok ? r.json() : [])
      .then(list => {
        const map = {};
        for (const p of list) map[p.id] = p;
        setProfiles(prev => ({ ...prev, ...map }));
      })
      .catch(() => {});
  }, [data]);

  useEffect(() => {
    if (chartView !== 'cal') return;
    if (range === 'custom' && (!customStart || !customEnd)) return;
    const controller = new AbortController();
    const q = buildQuery(getQueryParams(range, customStart, customEnd));
    setCalendarLoading(true);
    fetch(`${BASE}/api/daily-stats${q}`, { credentials: 'include', signal: controller.signal })
      .then(r => r.ok ? r.json() : [])
      .then(list => setCalendarData(Array.isArray(list) ? list : []))
      .catch(err => { if (err.name !== 'AbortError') setCalendarData([]); })
      .finally(() => setCalendarLoading(false));
    return () => controller.abort();
  }, [chartView, range, customStart, customEnd]);

  async function handleRemoveMod(memberID) {
    try {
      const res = await fetch(`${BASE}/api/removed-mods`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ member_id: memberID }),
      });
      if (res.ok) {
        setRemovedIds(prev => new Set([...prev, memberID]));
      }
    } catch (_) {}
  }

  function displayName(id) {
    return profiles[id]?.username || id;
  }

  // Filter removed mods from all lists
  const activeData = data.filter(d => !removedIds.has(d.member_id));
  const activeModIds = activeData.map(d => d.member_id);

  const metricKey = METRIC_KEY[activeTab];
  const sortedData = [...activeData].sort((a, b) => {
    const diff = (b[metricKey] ?? 0) - (a[metricKey] ?? 0);
    return sortDir === 'desc' ? diff : -diff;
  });

  const isMetricTab = METRIC_KEY[activeTab] !== undefined;

  return (
    <section className="section">
      <h2 className="section-title">Management</h2>

      <div className="tab-nav">
        {TABS.map(tab => (
          <button
            key={tab}
            className={`tab-btn${activeTab === tab ? ' active' : ''}`}
            onClick={() => { setActiveTab(tab); setSortDir('desc'); setChartView('bar'); }}
          >
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'Moderators' && (
        <ModeratorsTab
          modIds={activeModIds}
          profiles={profiles}
          setProfiles={setProfiles}
          removedIds={removedIds}
          onRemoveMod={handleRemoveMod}
        />
      )}

      {activeTab === 'Mod Notes' && (
        <ModNotesTab
          isDirector={isDirector}
          modIds={activeModIds}
          profiles={profiles}
          setProfiles={setProfiles}
        />
      )}

      {isMetricTab && (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12, marginBottom: 4 }}>
            <div className="date-range-btns" style={{ marginBottom: 0 }}>
              {presets.map(({ key, label }) => (
                <button
                  key={key}
                  className={`date-range-btn${range === key ? ' active' : ''}`}
                  onClick={() => setRange(key)}
                >
                  {label}
                </button>
              ))}
            </div>
            <ViewSwitcher view={chartView} setView={setChartView} />
          </div>

          {range === 'custom' && (
            <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginBottom: 20, flexWrap: 'wrap' }}>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">From</label>
                <input
                  type="date"
                  className="form-input"
                  style={{ padding: '6px 10px', fontSize: 13 }}
                  value={customStart}
                  onChange={e => setCustomStart(e.target.value)}
                />
              </div>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">To</label>
                <input
                  type="date"
                  className="form-input"
                  style={{ padding: '6px 10px', fontSize: 13 }}
                  value={customEnd}
                  onChange={e => setCustomEnd(e.target.value)}
                />
              </div>
            </div>
          )}

          {loading ? (
            <p className="loading-text">Loading...</p>
          ) : range === 'custom' && (!customStart || !customEnd) ? (
            <p className="loading-text">Select a date range to view data.</p>
          ) : (
            <>
              {chartView === 'bar' && (
                <BarChart
                  bars={sortedData.map(row => ({
                    label: displayName(row.member_id),
                    value: row[metricKey] ?? 0,
                    color: METRIC_COLOR[metricKey] ?? '#7289da',
                  }))}
                />
              )}
              {chartView === 'pie' && (
                <PieChart
                  slices={sortedData
                    .filter(row => (row[metricKey] ?? 0) > 0)
                    .map((row, i) => ({
                      label: displayName(row.member_id),
                      value: row[metricKey] ?? 0,
                      color: PIE_COLORS[i % PIE_COLORS.length],
                    }))}
                />
              )}
              {chartView === 'cal' && (
                calendarLoading
                  ? <p className="loading-text">Loading calendar...</p>
                  : <CalendarHeatmap dailyData={calendarData} baseColor={METRIC_COLOR[metricKey] ?? '#7289da'} />
              )}
              {chartView === 'num' && (
                sortedData.length === 0
                  ? <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No data for this period.</p>
                  : <table className="loa-table">
                      <thead>
                        <tr>
                          <th>Moderator</th>
                          <th style={{ textAlign: 'right' }}>{activeTab}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sortedData.map(row => (
                          <tr key={row.member_id}>
                            <td>{displayName(row.member_id)}</td>
                            <td style={{ textAlign: 'right', fontWeight: 600, color: METRIC_COLOR[metricKey] ?? 'var(--discord-text)' }}>
                              {(row[metricKey] ?? 0).toLocaleString()}{metricKey === 'voice' ? 'h' : ''}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
}
