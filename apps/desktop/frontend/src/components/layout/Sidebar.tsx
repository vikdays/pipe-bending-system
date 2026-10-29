import { Brand } from '../Brand';
import { Icon } from '../Icon';
import type { WorkspaceView } from './workspaceTypes';

export function Sidebar({
  view,
  taskCount,
  onNavigate,
}: {
  view: WorkspaceView;
  taskCount: number;
  onNavigate: (view: WorkspaceView) => void;
}) {
  return (
    <aside className="sidebar">
      <Brand compact />
      <div className="sidebar-caption">РАБОЧЕЕ МЕСТО</div>
      <nav className="main-nav" aria-label="Основная навигация">
        <button
          className={`nav-item ${view === 'overview' ? 'nav-item--active' : ''}`}
          aria-current={view === 'overview' ? 'page' : undefined}
          onClick={() => onNavigate('overview')}
        >
          <Icon name="grid" />
          <span>Обзор работы</span>
        </button>
        <button
          className={`nav-item ${view === 'tasks' ? 'nav-item--active' : ''}`}
          aria-current={view === 'tasks' ? 'page' : undefined}
          onClick={() => onNavigate('tasks')}
        >
          <Icon name="clipboard" />
          <span>Задания</span>
          <span className="nav-count">{taskCount}</span>
        </button>
        <button
          className="nav-item"
          disabled
          title="Трубный журнал будет реализован следующим этапом"
        >
          <Icon name="pipe" />
          <span>Трубный журнал</span>
        </button>
        <button className="nav-item" disabled title="Раздел ещё не реализован">
          <Icon name="sliders" />
          <span>Конфигурации</span>
        </button>
        <button className="nav-item" disabled title="Архив ещё не реализован">
          <Icon name="archive" />
          <span>Архив протоколов</span>
        </button>
      </nav>
      <div className="sidebar-bottom">
        <div className="offline-card">
          <div className="offline-card-icon">
            <Icon name="shield" size={20} />
          </div>
          <div>
            <strong>Автономная работа</strong>
            <p>ПК руководителя</p>
          </div>
        </div>
        <button
          className="nav-item"
          disabled
          title="Настройки ещё не реализованы"
        >
          <Icon name="settings" />
          <span>Настройки</span>
        </button>
        <div className="sidebar-version">
          Контроль гибки<span>Прототип 0.1</span>
        </div>
      </div>
    </aside>
  );
}
