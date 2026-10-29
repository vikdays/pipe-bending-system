import { Icon } from '../../components/Icon';
import type { TaskCounts, TaskFilter } from './taskTypes';

export function TaskSummaryCards({
  total,
  counts,
  onFilter,
}: {
  total: number;
  counts: TaskCounts;
  onFilter: (filter: TaskFilter) => void;
}) {
  return (
    <section className="stats-grid" aria-label="Сводка заданий">
      <button className="stat-card" onClick={() => onFilter('all')}>
        <span className="stat-label">
          Всего заданий
          <Icon name="clipboard" size={19} />
        </span>
        <strong>{total.toString().padStart(2, '0')}</strong>
        <span className="stat-caption">В рабочем списке</span>
      </button>
      <button className="stat-card" onClick={() => onFilter('ready')}>
        <span className="stat-label">
          Готовы к передаче
          <Icon name="upload" size={19} />
        </span>
        <strong>{counts.ready.toString().padStart(2, '0')}</strong>
        <span className="stat-caption">
          <span className="small-dot small-dot--blue" />
          Можно передать на планшет
        </span>
      </button>
      <button className="stat-card" onClick={() => onFilter('sent')}>
        <span className="stat-label">
          Ожидают результатов
          <Icon name="clock" size={19} />
        </span>
        <strong>{counts.sent.toString().padStart(2, '0')}</strong>
        <span className="stat-caption">
          <span className="small-dot small-dot--amber" />
          Переданы машинисту
        </span>
      </button>
      <button className="stat-card" onClick={() => onFilter('completed')}>
        <span className="stat-label">
          Завершены
          <Icon name="check" size={19} />
        </span>
        <strong>{counts.completed.toString().padStart(2, '0')}</strong>
        <span className="stat-caption">
          <span className="small-dot small-dot--green" />
          Изготовление завершено
        </span>
      </button>
    </section>
  );
}
