import { useState } from 'react';

// ---- View Switcher ----
// view: 'bar' | 'pie' | 'cal'
// setView: (v: string) => void
export function ViewSwitcher({ view, setView }) {
  const views = [
    { key: 'bar', label: '▌ Bar' },
    { key: 'pie', label: '◕ Pie' },
    { key: 'cal', label: '📅 Calendar' },
  ];
  return (
    <div style={{ display: 'flex', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, overflow: 'hidden' }}>
      {views.map(v => (
        <button
          key={v.key}
          onClick={() => setView(v.key)}
          style={{
            padding: '5px 14px', border: 'none', fontSize: 12, cursor: 'pointer',
            background: view === v.key ? 'var(--discord-blurple)' : 'var(--discord-card)',
            color: view === v.key ? '#fff' : 'var(--discord-muted)',
            borderLeft: v.key !== 'bar' ? '1px solid rgba(255,255,255,0.1)' : 'none',
          }}
        >
          {v.label}
        </button>
      ))}
    </div>
  );
}

// ---- Bar Chart ----
// bars: [{ label: string, value: number, color: string }]
export function BarChart({ bars = [] }) {
  if (!bars.length) return <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No data for this period.</p>;
  const maxVal = Math.max(...bars.map(b => b.value), 1);
  return (
    <div style={{ display: 'flex', alignItems: 'flex-end', gap: 10, height: 160, paddingBottom: 28 }}>
      {bars.map(bar => {
        const heightPx = Math.round((bar.value / maxVal) * 120);
        return (
          <div key={bar.label} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4, flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 11, color: 'var(--discord-text)' }}>{bar.value.toLocaleString()}</div>
            <div style={{
              background: bar.color, width: '100%', height: heightPx,
              borderRadius: '3px 3px 0 0', minHeight: 2,
            }} />
            <div style={{
              fontSize: 10, color: 'var(--discord-muted)',
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '100%',
            }}>
              {bar.label}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ---- Pie Chart ----
// slices: [{ label: string, value: number, color: string }]
export function PieChart({ slices = [] }) {
  const total = slices.reduce((s, x) => s + x.value, 0);
  if (total === 0) return <p style={{ color: 'var(--discord-muted)', fontSize: 14 }}>No data for this period.</p>;

  let acc = 0;
  const gradient = 'conic-gradient(' + slices.map(({ color, value }) => {
    const from = (acc / total * 100).toFixed(2);
    acc += value;
    const to = (acc / total * 100).toFixed(2);
    return `${color} ${from}% ${to}%`;
  }).join(',') + ')';

  return (
    <div style={{ display: 'flex', gap: 24, alignItems: 'center', flexWrap: 'wrap', padding: '10px 0' }}>
      <div style={{ width: 120, height: 120, borderRadius: '50%', background: gradient, flexShrink: 0 }} />
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {slices.map(s => (
          <div key={s.label} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13 }}>
            <div style={{ width: 12, height: 12, borderRadius: 2, background: s.color, flexShrink: 0 }} />
            <span style={{ color: 'var(--discord-text)' }}>
              {s.label} — {s.value.toLocaleString()} ({Math.round(s.value / total * 100)}%)
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ---- Calendar Heatmap ----
// dailyData: [{ date: 'YYYY-MM-DD', messages: number, tickets: number, qa: number, voice_hours: number }]
// baseColor: CSS hex color string, e.g. '#7289da'
const TOOLTIP_METRICS = [
  { label: 'Messages', key: 'messages',    color: '#7289da' },
  { label: 'Tickets',  key: 'tickets',     color: '#43b581' },
  { label: 'Q&A',      key: 'qa',          color: '#faa61a' },
  { label: 'Voice',    key: 'voice_hours', color: '#f04747', suffix: 'h' },
];

export function CalendarHeatmap({ dailyData, baseColor = '#7289da' }) {
  const [tooltip, setTooltip] = useState({ visible: false, x: 0, y: 0, data: null });
  const [month, setMonth] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });

  const dataByDate = {};
  for (const row of dailyData) dataByDate[row.date] = row;

  const maxActivity = Math.max(
    ...dailyData.map(d => (d.messages || 0) + (d.tickets || 0) + (d.qa || 0) + (d.voice_hours || 0)),
    1
  );

  const year = month.getFullYear();
  const mon = month.getMonth();
  const firstDay = new Date(year, mon, 1).getDay(); // 0 = Sunday
  const daysInMonth = new Date(year, mon + 1, 0).getDate();
  const offset = (firstDay + 6) % 7; // shift so Monday = col 0

  const cells = [];
  for (let i = 0; i < offset; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(d);

  const monthLabel = month.toLocaleString('default', { month: 'long', year: 'numeric' });

  // Parse hex to r,g,b for rgba()
  // Normalize baseColor to a 6-digit hex string; fall back to discord blurple
  const hexStr = /^#[0-9a-fA-F]{6}$/.test(baseColor) ? baseColor : '#7289da';
  const r = parseInt(hexStr.slice(1, 3), 16);
  const g = parseInt(hexStr.slice(3, 5), 16);
  const b = parseInt(hexStr.slice(5, 7), 16);

  return (
    <div style={{ position: 'relative' }}>
      {/* Month navigation */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 12 }}>
        <button
          className="btn btn-muted btn-sm"
          style={{ padding: '2px 10px' }}
          onClick={() => setMonth(m => new Date(m.getFullYear(), m.getMonth() - 1, 1))}
        >
          ‹
        </button>
        <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--discord-text)', minWidth: 150, textAlign: 'center' }}>
          {monthLabel}
        </span>
        <button
          className="btn btn-muted btn-sm"
          style={{ padding: '2px 10px' }}
          onClick={() => setMonth(m => new Date(m.getFullYear(), m.getMonth() + 1, 1))}
        >
          ›
        </button>
      </div>

      {/* Day-of-week headers */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)', gap: 4, maxWidth: 294, marginBottom: 4 }}>
        {['M','T','W','T','F','S','S'].map((d, i) => (
          <div key={i} style={{ fontSize: 9, color: 'var(--discord-muted)', textAlign: 'center' }}>{d}</div>
        ))}
      </div>

      {/* Calendar cells */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)', gap: 4, maxWidth: 294 }}>
        {cells.map((day, i) => {
          if (day === null) return <div key={`e-${i}`} />;
          const dateStr = `${year}-${String(mon + 1).padStart(2,'0')}-${String(day).padStart(2,'0')}`;
          const d = dataByDate[dateStr];
          const activity = d ? (d.messages || 0) + (d.tickets || 0) + (d.qa || 0) + (d.voice_hours || 0) : 0;
          const opacity = activity > 0 ? Math.max(0.12, (activity / maxActivity) * 0.88 + 0.07) : 0.06;
          return (
            <div
              key={dateStr}
              style={{
                background: `rgba(${r},${g},${b},${opacity})`,
                borderRadius: 3, aspectRatio: '1',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 9, color: 'var(--discord-text)',
              }}
              onMouseEnter={e => {
                if (!d) return;
                const tipW = 200;
                const x = e.clientX + 14 + tipW > window.innerWidth ? e.clientX - tipW - 4 : e.clientX + 14;
                setTooltip({ visible: true, x, y: e.clientY - 10, data: { dateStr, ...d } });
              }}
              onMouseMove={e => setTooltip(t => {
                if (!t.visible) return t;
                const tipW = 200;
                const x = e.clientX + 14 + tipW > window.innerWidth ? e.clientX - tipW - 4 : e.clientX + 14;
                return { ...t, x, y: e.clientY - 10 };
              })}
              onMouseLeave={() => setTooltip(t => ({ ...t, visible: false }))}
            >
              {day}
            </div>
          );
        })}
      </div>

      {/* Hover tooltip */}
      {tooltip.visible && tooltip.data && (
        <div style={{
          position: 'fixed', left: tooltip.x, top: tooltip.y, zIndex: 9999,
          background: 'var(--discord-dark)', border: '1px solid rgba(255,255,255,0.1)',
          borderRadius: 8, padding: '10px 14px', fontSize: 12, color: 'var(--discord-text)',
          pointerEvents: 'none', minWidth: 180, boxShadow: '0 4px 16px rgba(0,0,0,0.5)',
        }}>
          <div style={{ fontWeight: 700, marginBottom: 8, fontSize: 13 }}>{tooltip.data.dateStr}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {TOOLTIP_METRICS.map(({ label, key, color, suffix = '' }) => (
              <div key={key} style={{ display: 'flex', justifyContent: 'space-between', gap: 16 }}>
                <span style={{ color }}>{label}</span>
                <span style={{ fontWeight: 600 }}>{(tooltip.data[key] || 0).toLocaleString()}{suffix}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
