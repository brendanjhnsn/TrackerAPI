import { useState } from 'react';
import './App.css';
import { AuthProvider, useAuth } from './context/AuthContext';
import Header from './components/Header';
import TeamOverview from './components/TeamOverview';
import LoaForm from './components/LoaForm';
import ManagementPanel from './components/ManagementPanel';
import GameLeadsPage from './components/GameLeadsPage';
import ModeratorsPage from './components/ModeratorsPage';
import PermissionsPage from './components/PermissionsPage';
import AuditLogPage from './components/AuditLogPage';

function AppContent() {
  const { user } = useAuth();
  const isLoggedIn = user !== null && user !== undefined;
  const isManagement = user?.role === 'manager' || user?.role === 'director';
  const [currentPage, setCurrentPage] = useState('home');

  return (
    <div className="app">
      <Header currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <main className="main">
        {currentPage === 'game_leads' && user?.permissions?.game_leads ? (
          <GameLeadsPage />
        ) : currentPage === 'moderators' && user?.permissions?.moderators ? (
          <ModeratorsPage />
        ) : currentPage === 'permissions' && user?.role === 'director' ? (
          <PermissionsPage />
        ) : currentPage === 'audit_log' && user?.role === 'director' ? (
          <AuditLogPage />
        ) : (
          <>
            <TeamOverview />
            {isLoggedIn && <LoaForm />}
            {isManagement && user?.permissions?.management_panel && <ManagementPanel />}
          </>
        )}
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}
