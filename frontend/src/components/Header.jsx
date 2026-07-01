import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

export default function Header({ currentPage, setCurrentPage }) {
  const { user, logout } = useAuth();
  const isManagement = user?.role === 'manager' || user?.role === 'director';
  const isDirector = user?.role === 'director';

  return (
    <header className="header">
      <div className="header-left">
        <span
          className={`header-title${isManagement ? ' clickable' : ''}`}
          onClick={() => isManagement && setCurrentPage('home')}
        >
          Community Tracker
        </span>
        {isManagement && (
          <nav className="header-nav">
            {user?.permissions?.moderators && (
              <button
                className={`header-nav-btn${currentPage === 'moderators' ? ' active' : ''}`}
                onClick={() => setCurrentPage('moderators')}
              >
                Moderators
              </button>
            )}
            {user?.permissions?.game_leads && (
              <button
                className={`header-nav-btn${currentPage === 'game_leads' ? ' active' : ''}`}
                onClick={() => setCurrentPage('game_leads')}
              >
                Game Leads
              </button>
            )}
            {isDirector && (
              <>
                <button
                  className={`header-nav-btn${currentPage === 'audit_log' ? ' active' : ''}`}
                  onClick={() => setCurrentPage('audit_log')}
                >
                  Audit Log
                </button>
                <button
                  className={`header-nav-btn${currentPage === 'permissions' ? ' active' : ''}`}
                  onClick={() => setCurrentPage('permissions')}
                >
                  Permissions
                </button>
              </>
            )}
          </nav>
        )}
      </div>
      <div className="header-right">
        {user === undefined && (
          <span className="loading-text">Loading...</span>
        )}
        {user === null && (
          <a href={`${BASE}/auth/discord/redirect`} className="btn btn-blurple">
            Login with Discord
          </a>
        )}
        {user && (
          <>
            {user.avatar_url && (
              <img src={user.avatar_url} alt="" className="user-avatar" />
            )}
            <span className="user-chip">{user.username || user.discord_user_id}</span>
            <button className="btn btn-red" onClick={logout}>Logout</button>
          </>
        )}
      </div>
    </header>
  );
}
