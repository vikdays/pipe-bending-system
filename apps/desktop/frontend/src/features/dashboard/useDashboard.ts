import { useEffect, useState } from 'react';
import type { WorkspaceView } from '../../components/layout/workspaceTypes';
import type { TaskService } from '../tasks/taskService';
import type {
  PipeOption,
  Task,
  TaskCounts,
  TaskDraft,
  TaskFilter,
} from '../tasks/taskTypes';

type DashboardDialog =
  | { kind: 'create' }
  | { kind: 'task'; task: Task }
  | { kind: 'exchange' }
  | null;

export function useDashboard(service: TaskService) {
  const [view, setView] = useState<WorkspaceView>('overview');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [pipeOptions, setPipeOptions] = useState<PipeOption[]>([]);
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<TaskFilter>('all');
  const [dialog, setDialog] = useState<DashboardDialog>(null);
  const [toast, setToast] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  useEffect(() => {
    let active = true;
    Promise.all([service.getTasks(), service.getPipeOptions()])
      .then(([loadedTasks, loadedPipes]) => {
        if (!active) return;
        setTasks(loadedTasks);
        setPipeOptions(loadedPipes);
        setIsLoading(false);
      })
      .catch(() => {
        if (!active) return;
        setLoadError('Не удалось получить список заданий. Попробуйте ещё раз.');
        setIsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [service, loadAttempt]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(''), 5000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const normalizedQuery = query.trim().toLocaleLowerCase('ru-RU');
  const visibleTasks = tasks.filter(
    (task) =>
      (filter === 'all' || task.status === filter) &&
      [task.number, task.pipeNumber, task.bendNumber, task.machine].some(
        (value) => value.toLocaleLowerCase('ru-RU').includes(normalizedQuery),
      ),
  );
  const counts = tasks.reduce<TaskCounts>(
    (result, task) => {
      result[task.status] += 1;
      return result;
    },
    { draft: 0, ready: 0, sent: 0, completed: 0 },
  );

  function applyFilter(value: TaskFilter) {
    setFilter(value);
    setQuery('');
  }

  function resetFilters() {
    applyFilter('all');
  }

  function navigate(nextView: WorkspaceView) {
    setView(nextView);
    resetFilters();
  }

  function retryLoading() {
    setLoadError('');
    setIsLoading(true);
    setLoadAttempt((attempt) => attempt + 1);
  }

  function openCreate() {
    setCreateError('');
    setDialog({ kind: 'create' });
  }

  async function createTask(draft: TaskDraft) {
    if (isCreating) return;
    setIsCreating(true);
    setCreateError('');
    try {
      const task = await service.createTask(draft);
      setTasks((current) => [task, ...current]);
      setDialog(null);
      resetFilters();
      setToast(`Черновик ${task.number} добавлен в демонстрационный список`);
    } catch (error) {
      setCreateError(
        error instanceof Error
          ? error.message
          : 'Не удалось создать задание. Попробуйте ещё раз.',
      );
    } finally {
      setIsCreating(false);
    }
  }

  return {
    view,
    tasks,
    pipeOptions,
    query,
    filter,
    dialog,
    toast,
    counts,
    visibleTasks,
    isLoading,
    loadError,
    isCreating,
    createError,
    setQuery,
    setFilter,
    applyFilter,
    resetFilters,
    navigate,
    retryLoading,
    openCreate,
    createTask,
    openTask: (task: Task) => setDialog({ kind: 'task', task }),
    openExchange: () => setDialog({ kind: 'exchange' }),
    closeDialog: () => setDialog(null),
    dismissToast: () => setToast(''),
  };
}
