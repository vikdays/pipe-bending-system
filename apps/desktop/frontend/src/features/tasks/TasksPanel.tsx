import { TaskFilters } from './TaskFilters';
import { TasksTable } from './TasksTable';
import type { Task, TaskCounts, TaskFilter } from './taskTypes';

interface TasksPanelProps {
  tasks: Task[];
  total: number;
  counts: TaskCounts;
  query: string;
  filter: TaskFilter;
  isLoading: boolean;
  error: string;
  onQueryChange: (query: string) => void;
  onFilterChange: (filter: TaskFilter) => void;
  onSelectTask: (task: Task) => void;
  onResetFilters: () => void;
  onRetry: () => void;
}

export function TasksPanel({
  tasks,
  total,
  counts,
  query,
  filter,
  isLoading,
  error,
  onQueryChange,
  onFilterChange,
  onSelectTask,
  onResetFilters,
  onRetry,
}: TasksPanelProps) {
  return (
    <section
      className="panel tasks-panel"
      aria-labelledby="tasks-heading"
      aria-busy={isLoading}
    >
      <div className="panel-heading">
        <div>
          <h2 id="tasks-heading">Задания на изготовление</h2>
          <p>От подготовки до получения результатов</p>
        </div>
        <span className="panel-total">{total}</span>
      </div>
      <TaskFilters
        query={query}
        filter={filter}
        total={total}
        counts={counts}
        onQueryChange={onQueryChange}
        onFilterChange={onFilterChange}
      />
      {isLoading ? (
        <div className="empty-state" role="status">
          <p>Загрузка заданий…</p>
        </div>
      ) : error ? (
        <div className="empty-state" role="alert">
          <p>{error}</p>
          <button className="button button--secondary" onClick={onRetry}>
            Повторить загрузку
          </button>
        </div>
      ) : (
        <TasksTable
          tasks={tasks}
          onSelectTask={onSelectTask}
          onResetFilters={onResetFilters}
        />
      )}
      <div className="table-footer">
        <span>
          Показано {tasks.length} из {total}
        </span>
        <span>Данные для демонстрации</span>
      </div>
    </section>
  );
}
