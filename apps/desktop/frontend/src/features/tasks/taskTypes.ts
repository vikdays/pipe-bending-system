import type { Pipe } from '../pipes/pipeTypes';
export type TaskStatus = 'draft' | 'ready' | 'sent' | 'completed';
export type TaskFilter = 'all' | TaskStatus;
export type TaskCounts = Record<TaskStatus, number>;
export interface Task {
  id: string;
  number: string;
  pipeId: string;
  pipeRevision: number;
  pipeInventoryNumber: string;
  pipeNumber: string;
  pipeSnapshot: Pipe;
  dimensions: string;
  bendNumber: string;
  machine: string;
  targetAngle: string;
  status: TaskStatus;
  date: string;
}
export interface TaskDraft {
  number: string;
  pipeId: string;
  pipeRevision: number;
  bendNumber: string;
  machine: string;
  targetAngle: string;
}
