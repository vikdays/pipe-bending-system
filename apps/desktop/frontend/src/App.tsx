import { useState } from 'react';
import { LoginScreen } from './features/auth/LoginScreen';
import { DashboardScreen } from './features/dashboard/DashboardScreen';

export function App() {
  const [isSignedIn, setIsSignedIn] = useState(false);

  return isSignedIn ? (
    <DashboardScreen onSignOut={() => setIsSignedIn(false)} />
  ) : (
    <LoginScreen onSignIn={() => setIsSignedIn(true)} />
  );
}
