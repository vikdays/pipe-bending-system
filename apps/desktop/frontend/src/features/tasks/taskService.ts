import { normalizePipe } from '../pipes/pipeTypes.ts';
import { pipeDimensions } from '../pipes/pipeFields.ts';
import type { Authorize } from '../pipes/pipeService.ts';
import type {
  Task as TaskDTO,
  TaskInput,
} from '../../../bindings/changeme/internal/registry/models';
import type { Task, TaskDraft } from './taskTypes.ts';
export interface TaskService {
  getTasks(): Promise<Task[]>;
  createTask(draft: TaskDraft): Promise<Task>;
}
export interface TaskTransport {
  ListTasks(access: string): Promise<TaskDTO[] | null>;
  CreateTask(access: string, input: TaskInput): Promise<TaskDTO>;
}
function toTask(record: TaskDTO): Task {
  if (!['draft', 'ready', 'sent', 'completed'].includes(record.status))
    throw new Error('Неподдерживаемый статус задания.');
  const pipe = normalizePipe(record.pipeSnapshot);
  return {
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
  };
}
export function createTaskService(
  transport: TaskTransport,
  authorize: Authorize,
): TaskService {
  return {
    getTasks: async () =>
      ((await authorize((access) => transport.ListTasks(access))) ?? []).map(
        toTask,
      ),
    createTask: async (draft) =>
      toTask(await authorize((access) => transport.CreateTask(access, draft))),
  };
}
