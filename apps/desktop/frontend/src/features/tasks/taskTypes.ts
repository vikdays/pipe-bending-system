// View model used by the demo. This is not the database or exchange schema.
export type TaskStatus = 'draft' | 'ready' | 'sent' | 'completed';
export type TaskFilter = 'all' | TaskStatus;
export type TaskCounts = Record<TaskStatus, number>;

export interface Task {
  id: string;
  number: string;
  pipeNumber: string;
  dimensions: string;
  bendNumber: string;
  machine: string;
  targetAngle: string;
  status: TaskStatus;
  date: string;
}

export interface TaskDraft {
  number: string;
  pipeNumber: string;
  bendNumber: string;
  machine: string;
  targetAngle: string;
}

export interface PipeOption {
  number: string;
  dimensions: string;
}
