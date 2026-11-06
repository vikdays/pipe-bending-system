import type { Configuration, TaskDraft } from './taskTypes';
import type { Pipe } from '../pipes/pipeTypes';

export function matchesConfiguration(
  c: Configuration,
  task: TaskDraft,
  pipe?: Pipe,
): boolean {
  const angle = task.targetAngle.trim().replace(',', '.');
  return (
    (!task.machineId || c.machineId === task.machineId) &&
    (pipe?.diameterMm == null || c.diameterMm === pipe.diameterMm) &&
    (!task.bendType || !c.bendType || c.bendType === task.bendType) &&
    (!angle || !c.targetAngle || Number(c.targetAngle) === Number(angle))
  );
}
