import { normalizePipe } from '../pipes/pipeTypes.ts';
import { pipeDimensions } from '../pipes/pipeFields.ts';
import type { Authorize } from '../pipes/pipeService.ts';
import type {
  Task as TaskDTO,
  TaskInput,
  TaskBatchInput,
  Configuration as ConfigurationDTO,
  ConfigurationInput,
  Machine,
  ExchangeEvent,
  TaskExportSelection,
} from '../../../bindings/changeme/internal/registry/models';
import type { Task, TaskDraft, Configuration } from './taskTypes.ts';
import { AppError } from '../auth/appError.ts';
export interface TaskExportResult {
  outcome: 'cancelled' | 'saved' | 'written_unlogged';
  fileName: string;
  taskCount: number;
  message: string;
  exportedTasks: TaskExportSelection[];
}
export interface TaskService {
  exportTasks(tasks: TaskExportSelection[]): Promise<TaskExportResult>;
  getExchangeEvents(): Promise<ExchangeEvent[]>;
  getMachines(): Promise<Machine[]>;
  getTasks(): Promise<Task[]>;
  createTask(draft: TaskDraft): Promise<Task>;
  getTask(id: string): Promise<Task>;
  createTasks(requestId: string, drafts: TaskDraft[]): Promise<Task[]>;
  updateTask(id: string, revision: number, draft: TaskDraft): Promise<Task>;
  prepareTask(id: string, revision: number): Promise<Task>;
  getConfigurations(): Promise<Configuration[]>;
  saveConfiguration(
    input: ConfigurationInput,
    existing?: Pick<Configuration, 'id' | 'revision'>,
  ): Promise<Configuration>;
}
export interface TaskTransport {
  ExportTasks(
    access: string,
    tasks: TaskExportSelection[],
  ): Promise<{
    outcome: string;
    fileName: string;
    taskCount: number;
    message: string;
    exportedTasks: TaskExportSelection[] | null;
  }>;
  ListExchangeEvents(access: string): Promise<ExchangeEvent[] | null>;
  ListMachines(access: string): Promise<Machine[] | null>;
  ListTasks(access: string): Promise<TaskDTO[] | null>;
  CreateTask(access: string, input: TaskInput): Promise<TaskDTO>;
  GetTask(access: string, id: string): Promise<TaskDTO>;
  CreateTasks(access: string, input: TaskBatchInput): Promise<TaskDTO[] | null>;
  UpdateTask(
    access: string,
    id: string,
    revision: number,
    input: TaskInput,
  ): Promise<TaskDTO>;
  PrepareTask(access: string, id: string, revision: number): Promise<TaskDTO>;
  ListConfigurations(access: string): Promise<ConfigurationDTO[] | null>;
  CreateConfiguration(
    access: string,
    input: ConfigurationInput,
  ): Promise<ConfigurationDTO>;
  UpdateConfiguration(
    access: string,
    id: string,
    revision: number,
    input: ConfigurationInput,
  ): Promise<ConfigurationDTO>;
}
function toConfiguration(c: ConfigurationDTO): Configuration {
  return { ...c, parameters: c.parameters ?? [] };
}
function toTask(record: TaskDTO): Task {
  if (
    !['draft', 'ready', 'exported', 'sent', 'completed'].includes(record.status)
  )
    throw new Error('Неподдерживаемый статус задания.');
  const pipe = normalizePipe(record.pipeSnapshot);
  return {
    ...record,
    id: record.id,
    number: record.number,
    pipeId: record.pipeId,
    pipeRevision: record.pipeRevision,
    pipeInventoryNumber: pipe.inventoryNumber,
    pipeNumber: pipe.factoryNumber,
    pipeSnapshot: pipe,
    dimensions: pipeDimensions(pipe),
    bendNumber: record.bendNumber,
    machine: record.machine,
    targetAngle: record.targetAngle,
    status: record.status as Task['status'],
    date: new Intl.DateTimeFormat('ru-RU').format(new Date(record.createdAt)),
    configurationSnapshot: record.configurationSnapshot
      ? toConfiguration(record.configurationSnapshot)
      : null,
    revision: record.revision ?? 1,
    queuePosition: record.queuePosition ?? 0,
  };
}
export function createTaskService(
  transport: TaskTransport,
  authorize: Authorize,
): TaskService {
  return {
    exportTasks: async (tasks) => {
      const result = await authorize((a) => transport.ExportTasks(a, tasks));
      if (
        !result ||
        !['cancelled', 'saved', 'written_unlogged'].includes(result.outcome) ||
        (result.outcome !== 'cancelled' &&
          (result.taskCount !== tasks.length || !result.fileName))
      )
        throw new AppError(
          'INTERNAL_ERROR',
          'Не получен результат выгрузки задания. Проверьте журнал обмена.',
        );
      const confirmed = result.exportedTasks ?? [];
      if (
        result.outcome === 'saved' &&
        (confirmed.length !== tasks.length ||
          new Set(confirmed.map((t) => t.id)).size !== confirmed.length ||
          confirmed.some(
            (t) =>
              !tasks.some(
                (expected) =>
                  expected.id === t.id && expected.revision === t.revision,
              ),
          ))
      )
        throw new AppError(
          'INTERNAL_ERROR',
          'Не получено подтверждение статусов выгрузки. Обновите список заданий и проверьте журнал обмена.',
        );
      return {
        ...result,
        exportedTasks: confirmed,
        outcome: result.outcome as TaskExportResult['outcome'],
      };
    },
    getExchangeEvents: async () =>
      (await authorize((a) => transport.ListExchangeEvents(a))) ?? [],
    getMachines: async () =>
      (await authorize((a) => transport.ListMachines(a))) ?? [],
    getTasks: async () =>
      ((await authorize((access) => transport.ListTasks(access))) ?? []).map(
        toTask,
      ),
    createTask: async (draft) =>
      toTask(await authorize((access) => transport.CreateTask(access, draft))),
    getTask: async (id) =>
      toTask(await authorize((a) => transport.GetTask(a, id))),
    createTasks: async (requestId, tasks) => {
      const rows = await authorize((a) =>
        transport.CreateTasks(a, { requestId, tasks }),
      );
      if (!rows?.length)
        throw new AppError(
          'INTERNAL_ERROR',
          'Не получен результат создания заданий. Повторите сохранение.',
        );
      return rows.map(toTask);
    },
    updateTask: async (id, revision, input) =>
      toTask(
        await authorize((a) => transport.UpdateTask(a, id, revision, input)),
      ),
    prepareTask: async (id, revision) =>
      toTask(await authorize((a) => transport.PrepareTask(a, id, revision))),
    getConfigurations: async () =>
      ((await authorize((a) => transport.ListConfigurations(a))) ?? []).map(
        toConfiguration,
      ),
    saveConfiguration: async (input, existing) =>
      toConfiguration(
        await authorize((a) =>
          existing
            ? transport.UpdateConfiguration(
                a,
                existing.id,
                existing.revision,
                input,
              )
            : transport.CreateConfiguration(a, input),
        ),
      ),
  };
}
