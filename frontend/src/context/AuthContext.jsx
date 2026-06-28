import { createContext, useContext, useEffect, useState } from 'react';

const AuthContext = createContext(null);

const BASE = import.meta.env.VITE_API_BASE_URL ?? '';

export function AuthProvider({ children }) {
  // undefined = loading, null = logged out, object = logged in
  const [user, setUser] = useState(undefined);

  useEffect(() => {
    fetch(`${BASE}/api/me`, { credentials: 'include' })
      .then(r => (r.ok ? r.json() : null))
      .then(data => setUser(data))
      .catch(() => setUser(null));
  }, []);

  const logout = async () => {
    await fetch(`${BASE}/auth/discord/logout`, {
      method: 'DELETE',
      credentials: 'include',
    });
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
