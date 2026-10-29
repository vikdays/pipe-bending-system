import { statusLabels } from './taskPresentation';
import type { TaskStatus } from './taskTypes';

export function TaskStatusBadge({ status }: { status: TaskStatus }) {
  return (
    <span className={`status-badge status-badge--${status}`}>
      <span />
      {statusLabels[status]}
    </span>
  );
}
