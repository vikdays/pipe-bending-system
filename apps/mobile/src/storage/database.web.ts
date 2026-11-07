import type { TaskRepository } from './taskRepository';

export async function openTaskRepository(): Promise<TaskRepository> {
  throw new Error('Проверяйте локальную SQLite в Android-приложении. Браузерная версия для этого этапа не поддерживается.');
}
