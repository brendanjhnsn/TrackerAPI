import { useAuth } from '../context/AuthContext';

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

export default function Header() {
  const { user, logout } = useAuth();

  return (
    <header className="header">
      <span className="header-title">Community Tracker</span>
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
