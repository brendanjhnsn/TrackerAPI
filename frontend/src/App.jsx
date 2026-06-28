import './App.css';
import { AuthProvider, useAuth } from './context/AuthContext';
import Header from './components/Header';
import TeamOverview from './components/TeamOverview';
import MyStats from './components/MyStats';
import LoaForm from './components/LoaForm';
import ManagementPanel from './components/ManagementPanel';

function AppContent() {
  const { user } = useAuth();
  const isLoggedIn = user !== null && user !== undefined;
  const isManagement = user?.role === 'manager' || user?.role === 'director';

  return (
    <div className="app">
      <Header />
      <main className="main">
        <TeamOverview />
        {isLoggedIn && <MyStats />}
        {isLoggedIn && <LoaForm />}
        {isManagement && <ManagementPanel />}
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
