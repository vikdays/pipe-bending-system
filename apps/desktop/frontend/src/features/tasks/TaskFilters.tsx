import { Icon } from '../../components/Icon';
import type { TaskFilter, TaskCounts } from './taskTypes';

type Filter = TaskFilter;
const filters: { id: Filter; label: string }[] = [
  { id: 'all', label: 'Все' },
  { id: 'draft', label: 'Черновики' },
  { id: 'ready', label: 'Готовы к передаче' },
  { id: 'sent', label: 'Переданы' },
  { id: 'completed', label: 'Завершены' },
];

export function TaskFilters({
  query,
  filter,
  total,
  counts,
  onQueryChange,
  onFilterChange,
}: {
  query: string;
  filter: TaskFilter;
  total: number;
  counts: TaskCounts;
  onQueryChange: (query: string) => void;
  onFilterChange: (filter: TaskFilter) => void;
}) {
  return (
    <>
      <div className="table-toolbar">
        <div className="table-search">
          <Icon name="search" size={18} />
          <input
            aria-label="Поиск заданий"
            placeholder="Номер задания, трубы или отвода"
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
          />
          {query && (
            <button
              className="icon-button"
              aria-label="Очистить поиск"
              onClick={() => onQueryChange('')}
            >
              <Icon name="close" size={16} />
            </button>
          )}
        </div>
      </div>
      <div className="filter-tabs" aria-label="Фильтр по состоянию задания">
        {filters.map((item) => (
          <button
            key={item.id}
            className={`filter-tab ${filter === item.id ? 'filter-tab--active' : ''}`}
            aria-pressed={filter === item.id}
            onClick={() => onFilterChange(item.id)}
          >
            {item.label}
            <span>{item.id === 'all' ? total : counts[item.id]}</span>
          </button>
        ))}
      </div>
    </>
  );
}
