import type { Pipe } from '../pipes/pipeTypes';
import type {
  Task as TaskDTO,
  TaskInput,
  Configuration as ConfigurationDTO,
} from '../../../bindings/changeme/internal/registry/models';
export type Configuration = Omit<ConfigurationDTO, 'parameters'> & {
  parameters: NonNullable<ConfigurationDTO['parameters']>;
};
export type {
  ConfigurationInput,
  Machine,
} from '../../../bindings/changeme/internal/registry/models';
export type TaskStatus = 'draft' | 'ready' | 'exported' | 'sent' | 'completed';
export type TaskFilter = 'all' | TaskStatus;
export type TaskCounts = Record<TaskStatus, number>;
export interface Task extends Omit<
  TaskDTO,
  'pipeSnapshot' | 'status' | 'configurationSnapshot'
> {
  pipeInventoryNumber: string;
  pipeNumber: string;
  pipeSnapshot: Pipe;
  dimensions: string;
  configurationSnapshot: Configuration | null;
  status: TaskStatus;
  date: string;
}
export type TaskDraft = TaskInput;
export function emptyTaskDraft(): TaskDraft {
  return {
    number: '',
    pipeId: '',
    pipeRevision: 0,
    bendNumber: '',
    machine: '',
    machineId: '',
    targetAngle: '',
    bendType: '',
    notes: '',
    configurationId: '',
    configurationRevision: 0,
  };
}
