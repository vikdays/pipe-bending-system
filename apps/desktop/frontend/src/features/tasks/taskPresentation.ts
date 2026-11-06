import type { TaskStatus } from './taskTypes';

export const statusLabels: Record<TaskStatus, string> = {
  draft: 'Черновик',
  ready: 'Готово к передаче',
  exported: 'Выгружено',
  sent: 'Передано',
  completed: 'Завершено',
};
