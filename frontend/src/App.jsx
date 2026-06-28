import { useState } from 'react';
import './App.css';
import { AuthProvider, useAuth } from './context/AuthContext';
import Header from './components/Header';
import TeamOverview from './components/TeamOverview';
import LoaForm from './components/LoaForm';
import ManagementPanel from './components/ManagementPanel';
import ModeratorsPage from './components/ModeratorsPage';

function AppContent() {
  const { user } = useAuth();
  const isLoggedIn = user !== null && user !== undefined;
  const isManagement = user?.role === 'manager' || user?.role === 'director';
  const [currentPage, setCurrentPage] = useState('home');

  return (
    <div className="app">
      <Header currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <main className="main">
        {currentPage === 'moderators' && isManagement ? (
          <ModeratorsPage />
        ) : (
          <>
            <TeamOverview />
            {isLoggedIn && <LoaForm />}
            {isManagement && <ManagementPanel />}
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
