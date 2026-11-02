import { useState } from 'react';
import { LoginScreen } from './features/auth/LoginScreen';
import { DashboardScreen } from './features/dashboard/DashboardScreen';
import { useAuth } from './features/auth/useAuth';
import { AccountModal } from './features/auth/AccountModal';

export function App() {
  const auth = useAuth();
  const [accountOpen, setAccountOpen] = useState(false);

  if (auth.state.kind === 'loading')
    return (
      <main className="auth-state" role="status">
        Открываем локальную базу…
      </main>
    );
  if (auth.state.kind === 'unavailable')
    return (
      <main className="auth-state">
        <h1>Не удалось подключиться к приложению</h1>
        <p>
          Откройте настольное приложение через Wails. Если оно уже открыто,
          проверьте доступ к локальной базе данных.
        </p>
        <button className="button button--primary" onClick={auth.retry}>
          Повторить подключение
        </button>
      </main>
    );
  if (auth.state.kind !== 'signedIn')
    return (
      <LoginScreen
        key={auth.state.kind}
        setup={auth.state.kind === 'setup'}
        message={
          auth.state.kind === 'signedOut' ? auth.state.message : undefined
        }
        messageIsError={auth.state.kind === 'signedOut' && auth.state.isError}
        onSignIn={async (username, password, setup) => {
          setAccountOpen(false);
          await auth.signIn(username, password, setup);
        }}
      />
    );

  const user = auth.state.user;
  const signOut = () => {
    setAccountOpen(false);
    void auth.signOut();
  };
  return (
    <>
      {auth.state.warning && (
        <p className="auth-session-warning" role="alert">
          {auth.state.warning}
        </p>
      )}
      {user.role === 'admin' ? (
        <DashboardScreen
          user={user}
          onOpenAccount={() => setAccountOpen(true)}
          onSignOut={signOut}
        />
      ) : (
        <main className="auth-state">
          <h1>Выполнен вход: {user.username}</h1>
          <p>
            Рабочее место руководителя доступно пользователям с ролью
            «Руководитель работ». Для машиниста предусмотрено приложение
            Android.
          </p>
          <button
            className="button button--secondary"
            onClick={() => setAccountOpen(true)}
          >
            Учётная запись
          </button>
          <button className="button button--primary" onClick={signOut}>
            Выйти из системы
          </button>
        </main>
      )}
      {accountOpen && (
        <AccountModal user={user} onClose={() => setAccountOpen(false)} />
      )}
    </>
  );
}
