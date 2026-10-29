import type { ReactNode } from 'react';
import { Icon } from '../Icon';
import { Sidebar } from './Sidebar';
import type { WorkspaceView } from './workspaceTypes';

export function AppShell({
  view,
  taskCount,
  onNavigate,
  onSignOut,
  children,
}: {
  view: WorkspaceView;
  taskCount: number;
  onNavigate: (view: WorkspaceView) => void;
  onSignOut: () => void;
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
            <strong>{view === 'overview' ? 'Обзор' : 'Задания'}</strong>
          </div>
          <div className="topbar-user">
            <span className="avatar">РР</span>
            <div>
              <strong>Руководитель работ</strong>
              <span>Демонстрационный вход</span>
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
