import React, { useState, useEffect } from "react";

const ModeratorStats = () => {
  const [stats, setStats] = useState({
    checks: [],
    messages: [],
    voiceTime: [],
    tickets: [],
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filterMode, setFilterMode] = useState("single"); // 'single' or 'range'
  const [selectedDate, setSelectedDate] = useState(
    new Date().toISOString().split("T")[0],
  );
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState(
    new Date().toISOString().split("T")[0],
  );
  const [selectedMemberId, setSelectedMemberId] = useState("");

  const API_BASE_URL = "http://localhost:8080";

  useEffect(() => {
    fetchAllStats();
  }, [filterMode, selectedDate, startDate, endDate, selectedMemberId]);

  const fetchAllStats = async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();

      if (filterMode === "single") {
        if (selectedDate) params.append("date", selectedDate);
      } else {
        if (startDate) params.append("start_date", startDate);
        if (endDate) params.append("end_date", endDate);
      }

      if (selectedMemberId) params.append("member_id", selectedMemberId);
      const queryString = params.toString();

      const [checksRes, messagesRes, voiceRes, ticketsRes] = await Promise.all([
        fetch(
          `${API_BASE_URL}/api/checks${queryString ? `?${queryString}` : ""}`,
        ),
        fetch(
          `${API_BASE_URL}/api/messages${queryString ? `?${queryString}` : ""}`,
        ),
        fetch(
          `${API_BASE_URL}/api/voicetime${queryString ? `?${queryString}` : ""}`,
        ),
        fetch(
          `${API_BASE_URL}/api/tickets${queryString ? `?${queryString}` : ""}`,
        ),
      ]);

      if (!checksRes.ok || !messagesRes.ok || !voiceRes.ok || !ticketsRes.ok) {
        throw new Error("Failed to fetch stats");
      }

      const [checksData, messagesData, voiceData, ticketsData] =
        await Promise.all([
          checksRes.json(),
          messagesRes.json(),
          voiceRes.json(),
          ticketsRes.json(),
        ]);

      setStats({
        checks: Array.isArray(checksData)
          ? checksData
          : checksData
            ? [checksData]
            : [],
        messages: Array.isArray(messagesData)
          ? messagesData
          : messagesData
            ? [messagesData]
            : [],
        voiceTime: Array.isArray(voiceData)
          ? voiceData
          : voiceData
            ? [voiceData]
            : [],
        tickets: Array.isArray(ticketsData)
          ? ticketsData
          : ticketsData
            ? [ticketsData]
            : [],
      });
    } catch (err) {
      setError(err.message);
      console.error("Error fetching stats:", err);
    } finally {
      setLoading(false);
    }
  };

  const formatVoiceTime = (seconds) => {
    if (!seconds) return "0h 0m";
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };

  const renderStatTable = (title, data, columns) => {
    if (!data || data.length === 0) {
      return (
        <div className="stat-section">
          <h3>{title}</h3>
          <p className="no-data">No data available</p>
        </div>
      );
    }

    return (
      <div className="stat-section">
        <h3>{title}</h3>
        <table className="stat-table">
          <thead>
            <tr>
              {columns.map((col) => (
                <th key={col.key}>{col.label}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.map((row, idx) => (
              <tr key={idx}>
                {columns.map((col) => (
                  <td key={col.key}>
                    {col.format ? col.format(row[col.key]) : row[col.key]}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  return (
    <div className="moderator-stats">
      <h1>Moderator Activity Statistics</h1>

      {error && <div className="error-message">{error}</div>}

      <div className="filters">
        <div className="filter-mode">
          <label>Filter Mode:</label>
          <select
            value={filterMode}
            onChange={(e) => setFilterMode(e.target.value)}
          >
            <option value="single">Single Date</option>
            <option value="range">Date Range</option>
          </select>
        </div>

        {filterMode === "single" ? (
          <div className="filter-group">
            <label htmlFor="date-filter">Date:</label>
            <input
              id="date-filter"
              type="date"
              value={selectedDate}
              onChange={(e) => setSelectedDate(e.target.value)}
            />
          </div>
        ) : (
          <>
            <div className="filter-group">
              <label htmlFor="start-date-filter">Start Date:</label>
              <input
                id="start-date-filter"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
            </div>
            <div className="filter-group">
              <label htmlFor="end-date-filter">End Date:</label>
              <input
                id="end-date-filter"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
              />
            </div>
          </>
        )}

        <div className="filter-group">
          <label htmlFor="member-filter">Member ID (optional):</label>
          <input
            id="member-filter"
            type="text"
            placeholder="Leave blank for all mods"
            value={selectedMemberId}
            onChange={(e) => setSelectedMemberId(e.target.value)}
          />
        </div>
      </div>

      {loading ? (
        <div className="loading">Loading statistics...</div>
      ) : (
        <div className="stats-container">
          {renderStatTable("Questions Answered", stats.checks, [
            { key: "date", label: "Date" },
            { key: "member_id", label: "Member ID" },
            { key: "count", label: "Checkmarks" },
          ])}

          {renderStatTable("Daily Messages", stats.messages, [
            { key: "date", label: "Date" },
            { key: "member_id", label: "Member ID" },
            { key: "count", label: "Messages" },
          ])}

          {renderStatTable("Voice Chat Time", stats.voiceTime, [
            { key: "date", label: "Date" },
            { key: "member_id", label: "Member ID" },
            {
              key: "total_seconds",
              label: "Duration",
              format: formatVoiceTime,
            },
            { key: "hours", label: "Hours" },
            { key: "minutes", label: "Minutes" },
          ])}

          {renderStatTable("Tickets Answered", stats.tickets, [
            { key: "date", label: "Date" },
            { key: "member_id", label: "Member ID" },
            { key: "tickets", label: "First Responses" },
          ])}
        </div>
      )}

      <style>{`
        .moderator-stats {
          padding: 20px;
          font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
          max-width: 1200px;
          margin: 0 auto;
        }

        h1 {
          color: #333;
          margin-bottom: 30px;
        }

        .filters {
          display: flex;
          padding: 15px;
          background: #f5f5f5;
          border-radius: 8px;
        }

        .filter-mode {
          display: flex;
          gap: 10px;
          align-items: center;
        }

        .filter-mode select,
        .filter-group input {
          padding: 8px 12px;
          border: 1px solid #ddd;
          border-radius: 4px;
          font-size: 14px;
        }

        .filter-mode select:focus,
        .filter-group input:focus {
          outline: none;
          border-color: #007bff;
          box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
          gap: 15px;
          margin-bottom: 30px;
          flex-wrap: wrap;
          align-items: flex-end;
        }

        .filter-group {
          display: flex;
          flex-direction: column;
          gap: 5px;
        }

        .filter-group label {
          font-weight: 600;
          color: #555;
          font-size: 14px;
        }

        .filter-group input {
          padding: 8px 12px;
          border: 1px solid #ddd;
          border-radius: 4px;
          font-size: 14px;
        }

        button {
          padding: 8px 16px;
          background-color: #007bff;
          color: white;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-weight: 600;
          transition: background-color 0.3s;
        }

        button:hover:not(:disabled) {
          background-color: #0056b3;
        }

        button:disabled {
          background-color: #ccc;
          cursor: not-allowed;
        }

        .error-message {
          padding: 12px;
          background-color: #f8d7da;
          color: #721c24;
          border: 1px solid #f5c6cb;
          border-radius: 4px;
          margin-bottom: 20px;
        }

        .loading {
          text-align: center;
          padding: 40px;
          color: #666;
          font-size: 16px;
        }

        .stats-container {
          display: grid;
          grid-template-columns: 1fr;
          gap: 30px;
        }

        .stat-section {
          background-color: #f9f9f9;
          border: 1px solid #ddd;
          border-radius: 6px;
          padding: 20px;
        }

        .stat-section h3 {
          margin-top: 0;
          margin-bottom: 15px;
          color: #333;
          font-size: 18px;
          border-bottom: 2px solid #007bff;
          padding-bottom: 10px;
        }

        .stat-table {
          width: 100%;
          border-collapse: collapse;
        }

        .stat-table thead {
          background-color: #007bff;
          color: white;
        }

        .stat-table th {
          padding: 12px;
          text-align: left;
          font-weight: 600;
        }

        .stat-table td {
          padding: 12px;
          border-bottom: 1px solid #e0e0e0;
        }

        .stat-table tbody tr:hover {
          background-color: #f0f0f0;
        }

        .stat-table tbody tr:nth-child(even) {
          background-color: #fafafa;
        }

        .no-data {
          color: #999;
          font-style: italic;
          padding: 20px;
          text-align: center;
        }

        @media (max-width: 768px) {
          .moderator-stats {
            padding: 10px;
          }

          .filters {
            flex-direction: column;
          }

          .stat-table {
            font-size: 12px;
          }

          .stat-table th,
          .stat-table td {
            padding: 8px;
          }
        }
      `}</style>
    </div>
  );
};

export default ModeratorStats;
