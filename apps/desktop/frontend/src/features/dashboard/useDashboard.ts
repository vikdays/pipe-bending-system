import { useEffect, useState } from 'react';
import type { WorkspaceView } from '../../components/layout/workspaceTypes';
import type { TaskService, TaskExportResult } from '../tasks/taskService';
import type { PipeService } from '../pipes/pipeService';
import type { Pipe } from '../pipes/pipeTypes';
import { applyExportedTaskStates } from '../tasks/taskExportState';
import { authErrorMessage } from '../auth/appError';
import type {
  Task,
  TaskCounts,
  TaskDraft,
  TaskFilter,
  Configuration,
  ConfigurationInput,
  Machine,
} from '../tasks/taskTypes';

type DashboardDialog =
  | { kind: 'create'; task?: Task; reviewConfiguration?: boolean }
  | { kind: 'task'; task: Task }
  | { kind: 'exchange' }
  | { kind: 'export' }
  | { kind: 'cards' }
  | null;

// хук, который хранит состояние рабочего экрана и загружает нужные данные при открытии рабочего экрана.
export function useDashboard(service: TaskService, pipeService: PipeService) {
  const [view, setView] = useState<WorkspaceView>('overview');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [pipeOptions, setPipeOptions] = useState<Pipe[]>([]);
  const [machines, setMachines] = useState<Machine[]>([]);
  const [configurations, setConfigurations] = useState<Configuration[]>([]);
  const [taskBusy, setTaskBusy] = useState(false);
  const [taskError, setTaskError] = useState('');
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
    Promise.all([
      service.getTasks(),
      pipeService.getPipes(),
      service.getConfigurations(),
      service.getMachines(),
    ])
      .then(
        ([loadedTasks, loadedPipes, loadedConfigurations, loadedMachines]) => {
          if (!active) return;
          setTasks(loadedTasks);
          setPipeOptions(loadedPipes);
          setConfigurations(loadedConfigurations);
          setMachines(loadedMachines);
          setIsLoading(false);
        },
      )
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
    { draft: 0, ready: 0, exported: 0, sent: 0, completed: 0 },
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

  async function createTask(requestId: string, drafts: TaskDraft[]) {
    if (isCreating) return;
    setIsCreating(true);
    setCreateError('');
    try {
      const added =
        dialog?.kind === 'create' && dialog.task
          ? [
              await service.updateTask(
                dialog.task.id,
                dialog.task.revision,
                drafts[0],
              ),
            ]
          : await service.createTasks(requestId, drafts);
      setTasks((current) =>
        [
          ...current.filter(
            (task) => !added.some((next) => next.id === task.id),
          ),
          ...added,
        ].sort((a, b) => a.queuePosition - b.queuePosition),
      );
      setDialog(null);
      resetFilters();
      setToast(
        added.length === 1
          ? `Черновик ${added[0].number} сохранён`
          : `Сохранено черновиков: ${added.length}`,
      );
    } catch (error) {
      setCreateError(authErrorMessage(error));
    } finally {
      setIsCreating(false);
    }
  }

  async function openTask(task: Task) {
    if (taskBusy) return;
    setTaskBusy(true);
    setTaskError('');
    setDialog({ kind: 'task', task });
    try {
      const current = await service.getTask(task.id);
      setDialog({ kind: 'task', task: current });
      setTasks((previous) =>
        previous.map((t) => (t.id === current.id ? current : t)),
      );
    } catch (e) {
      setTaskError(authErrorMessage(e));
    } finally {
      setTaskBusy(false);
    }
  }
  async function editTask(task: Task, reviewConfiguration = false) {
    if (taskBusy) return;
    setTaskBusy(true);
    setTaskError('');
    try {
      const [current, pipes, configs, loadedMachines] = await Promise.all([
        service.getTask(task.id),
        pipeService.getPipes(),
        service.getConfigurations(),
        service.getMachines(),
      ]);
      if (current.status !== 'draft') {
        setDialog({ kind: 'task', task: current });
        setTaskError('Подготовленное задание нельзя изменять.');
        return;
      }
      if (
        reviewConfiguration &&
        !configs.some((c) => c.id === current.configurationId)
      ) {
        setTaskError(
          'Конфигурация не найдена. Откройте изменение черновика и выберите конфигурацию.',
        );
        return;
      }
      setPipeOptions(pipes);
      setConfigurations(configs);
      setMachines(loadedMachines);
      setCreateError('');
      setDialog({ kind: 'create', task: current, reviewConfiguration });
    } catch (e) {
      setTaskError(authErrorMessage(e));
    } finally {
      setTaskBusy(false);
    }
  }
  async function prepareTask(task: Task) {
    if (taskBusy) return;
    setTaskBusy(true);
    setTaskError('');
    try {
      const prepared = await service.prepareTask(task.id, task.revision);
      setTasks((previous) =>
        previous.map((t) => (t.id === prepared.id ? prepared : t)),
      );
      setDialog({ kind: 'task', task: prepared });
      setToast(`Задание ${prepared.number} подготовлено к передаче`);
    } catch (e) {
      setTaskError(authErrorMessage(e));
    } finally {
      setTaskBusy(false);
    }
  }
  async function saveConfiguration(
    input: ConfigurationInput,
    existing?: Configuration,
  ) {
    const c = await service.saveConfiguration(input, existing);
    setConfigurations((previous) =>
      [...previous.filter((item) => item.id !== c.id), c].sort((a, b) =>
        a.name.localeCompare(b.name, 'ru'),
      ),
    );
    return c;
  }

  return {
    view,
    tasks,
    pipeOptions,
    configurations,
    machines,
    taskBusy,
    taskError,
    editTask,
    reviewTaskConfiguration: (task: Task) => editTask(task, true),
    prepareTask,
    openExport: () => setDialog({ kind: 'export' }),
    exportSaved: (result: TaskExportResult) => {
      setTasks((current) =>
        applyExportedTaskStates(current, result.exportedTasks),
      );
      setDialog(null);
      setToast(
        `Пакет сохранён: ${result.fileName}. Заданий выгружено: ${result.taskCount}. Получение планшетом не подтверждено.`,
      );
    },
    saveConfiguration,
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
    openTask,
    openCards: () => setDialog({ kind: 'cards' }),
    refreshConfigurations: async () =>
      setConfigurations(await service.getConfigurations()),
    openExchange: () => setDialog({ kind: 'exchange' }),
    closeDialog: () => {
      if (!taskBusy && !isCreating) setDialog(null);
    },
    dismissToast: () => setToast(''),
  };
}
