import { Icon } from '../../components/Icon';
import { TaskStatusBadge } from './TaskStatusBadge';
import type { Task } from './taskTypes';

export function TasksTable({
  tasks,
  onSelectTask,
  onResetFilters,
}: {
  tasks: Task[];
  onSelectTask: (task: Task) => void;
  onResetFilters: () => void;
}) {
  return (
    <div className="table-scroll">
      <table className="task-table">
        <thead>
          <tr>
            <th scope="col">Задание</th>
            <th scope="col">Исходная труба</th>
            <th scope="col">Отвод / станок</th>
            <th scope="col">Цель</th>
            <th scope="col">Состояние</th>
            <th scope="col">
              <span className="sr-only">Действие</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((task) => (
            <tr key={task.id}>
              <td>
                <button
                  className="table-link"
                  onClick={() => onSelectTask(task)}
                >
                  {task.number}
                </button>
                <span className="cell-secondary">{task.date}</span>
              </td>
              <td>
                <strong className="cell-primary">{task.pipeNumber}</strong>
                <span className="cell-secondary">{task.dimensions}</span>
              </td>
              <td>
                <span className="cell-primary">{task.bendNumber}</span>
                <span className="cell-secondary">{task.machine}</span>
              </td>
              <td>
                <span className="angle-value">
                  {task.targetAngle}
                  <span>°</span>
                </span>
              </td>
              <td>
                <TaskStatusBadge status={task.status} />
              </td>
              <td>
                <button
                  className="icon-button table-open"
                  aria-label={`Открыть ${task.number}`}
                  onClick={() => onSelectTask(task)}
                >
                  <Icon name="chevron" size={18} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {tasks.length === 0 && (
        <div className="empty-state">
          <span>
            <Icon name="search" size={28} />
          </span>
          <h3>Задания не найдены</h3>
          <p>Попробуйте изменить запрос или выбрать другое состояние.</p>
          <button className="button button--secondary" onClick={onResetFilters}>
            Сбросить фильтры
          </button>
        </div>
      )}
    </div>
  );
}
