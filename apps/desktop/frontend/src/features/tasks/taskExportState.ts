import type { Task } from './taskTypes';
import type { TaskExportSelection } from '../../../bindings/changeme/internal/registry/models';

// Apply only the exact revisions confirmed by Go. New tasks prepared while a
// save dialog was open remain in the next export queue.
export function applyExportedTaskStates(
  tasks: Task[],
  exported: TaskExportSelection[],
): Task[] {
  const revisions = new Map(exported.map((t) => [t.id, t.revision]));
  return tasks.map((task) =>
    task.status === 'ready' && revisions.get(task.id) === task.revision
      ? { ...task, status: 'exported' }
      : task,
  );
}
