import { demoTasks } from './demoTasks.ts';
import type { PipeOption, Task, TaskDraft } from './taskTypes.ts';

// Async boundary for a future Go/Wails implementation; no React dependencies.
export interface TaskService {
  getTasks(): Promise<Task[]>;
  getPipeOptions(): Promise<PipeOption[]>;
  createTask(draft: TaskDraft): Promise<Task>;
}

// One independent in-memory list per mounted dashboard. No persistent storage.
export function createDemoTaskService(): TaskService {
  let tasks = demoTasks.map((task) => ({ ...task }));
  const pipes = Array.from(
    new Map(
      demoTasks.map((task) => [
        task.pipeNumber,
        { number: task.pipeNumber, dimensions: task.dimensions },
      ]),
    ).values(),
  );

  return {
    async getTasks() {
      return tasks.map((task) => ({ ...task }));
    },
    async getPipeOptions() {
      return pipes.map((pipe) => ({ ...pipe }));
    },
    async createTask(draft) {
      const number = draft.number.trim();
      const bendNumber = draft.bendNumber.trim();
      const machine = draft.machine.trim();
      const targetAngle = draft.targetAngle.trim().replace(',', '.');
      const pipe = pipes.find((item) => item.number === draft.pipeNumber);

      if (!number || !bendNumber || !machine) {
        throw new Error(
          'Заполните номер задания, номер отвода и модель станка.',
        );
      }
      if (!pipe) {
        throw new Error('Выберите трубу из демонстрационного списка.');
      }
      if (!/^\d+(\.\d+)?$/.test(targetAngle)) {
        throw new Error('Введите целевой угол числом, например 9 или 9,5.');
      }

      // Only input syntax is checked. Production compatibility and limits are not defined.
      const task: Task = {
        id: crypto.randomUUID(),
        number,
        pipeNumber: pipe.number,
        dimensions: pipe.dimensions,
        bendNumber,
        machine,
        targetAngle,
        status: 'draft',
        date: new Intl.DateTimeFormat('ru-RU').format(new Date()),
      };
      tasks = [{ ...task }, ...tasks];
      return task;
    },
  };
}
