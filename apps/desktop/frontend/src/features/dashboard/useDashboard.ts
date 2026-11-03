import { useEffect, useState } from 'react';
import type { WorkspaceView } from '../../components/layout/workspaceTypes';
import type { TaskService } from '../tasks/taskService';
import type { PipeService } from '../pipes/pipeService';
import type { Pipe } from '../pipes/pipeTypes';
import { authErrorMessage } from '../auth/appError';
import type {
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

export function useDashboard(service: TaskService, pipeService: PipeService) {
  const [view, setView] = useState<WorkspaceView>('overview');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [pipeOptions, setPipeOptions] = useState<Pipe[]>([]);
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
    Promise.all([service.getTasks(), pipeService.getPipes()])
      .then(([loadedTasks, loadedPipes]) => {
        if (!active) return;
        setTasks(loadedTasks);
        setPipeOptions(loadedPipes);
        setIsLoading(false);
      })
      .catch((error: unknown) => {
        if (!active) return;
        setLoadError(authErrorMessage(error));
        setIsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [service, pipeService, loadAttempt]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(''), 5000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const normalizedQuery = query.trim().toLocaleLowerCase('ru-RU');
  const visibleTasks = tasks.filter(
    (task) =>
      (filter === 'all' || task.status === filter) &&
      [
        task.number,
        task.pipeNumber,
        task.pipeInventoryNumber,
        task.bendNumber,
        task.machine,
      ].some((value) =>
        value.toLocaleLowerCase('ru-RU').includes(normalizedQuery),
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
      setToast(`Черновик ${task.number} сохранён`);
    } catch (error) {
      setCreateError(authErrorMessage(error));
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
