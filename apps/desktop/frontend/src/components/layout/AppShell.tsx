import type { ReactNode } from 'react';
import { Icon } from '../Icon';
import { Sidebar } from './Sidebar';
import type { WorkspaceView } from './workspaceTypes';
import type { AuthUser } from '../../features/auth/authClientCore';

export function AppShell({
  view,
  taskCount,
  onNavigate,
  onSignOut,
  user,
  onOpenAccount,
  children,
}: {
  view: WorkspaceView;
  taskCount: number;
  onNavigate: (view: WorkspaceView) => void;
  onSignOut: () => void;
  user: AuthUser;
  onOpenAccount: () => void;
  children: ReactNode;
}) {
  return (
    <div className="app-shell">
      <Sidebar view={view} taskCount={taskCount} onNavigate={onNavigate} />
      <div className="workspace">
        <header className="topbar">
          <div className="breadcrumbs">
            <span>Рабочее место</span>
            <Icon name="chevron" size={14} />
            <strong>
              {view === 'overview'
                ? 'Обзор'
                : view === 'pipes'
                  ? 'Трубный журнал'
                  : 'Задания'}
            </strong>
          </div>
          <div className="topbar-user">
            <button
              className="avatar"
              type="button"
              onClick={onOpenAccount}
              aria-label="Открыть учётную запись"
            >
              {user.username.slice(0, 2).toUpperCase()}
            </button>
            <div>
              <strong>{user.username}</strong>
              <span>Руководитель работ</span>
            </div>
            <span className="topbar-divider" />
            <button
              className="icon-button signout"
              aria-label="Выйти из системы"
              title="Выйти из системы"
              onClick={onSignOut}
            >
              <Icon name="logout" />
            </button>
          </div>
        </header>
        <main className="workspace-main">
          {children}
          <footer className="workspace-footer">
            <span>Цифровой контроль изготовления отводов</span>
            <span>Рабочее место руководителя</span>
          </footer>
        </main>
      </div>
    </div>
  );
}
