import type { TaskStatus } from './taskTypes';

export const statusLabels: Record<TaskStatus, string> = {
  draft: 'Черновик',
  ready: 'Готово к передаче',
  sent: 'Передано',
  completed: 'Завершено',
};
