import { canonicalJson, parseTaskPackage } from '../features/tasks/taskPackage.ts';
import type { ImportResult, TaskSummary, TransferredTask } from '../features/tasks/taskTypes.ts';
import { taskSchema } from './migrations/001-tasks.ts';

type Value = string | number | null;
export interface SqlDriver {
  execAsync(sql: string): Promise<void>;
  runAsync(sql: string, ...values: Value[]): Promise<unknown>;
  getFirstAsync<T>(sql: string, ...values: Value[]): Promise<T | null>;
  getAllAsync<T>(sql: string, ...values: Value[]): Promise<T[]>;
}
type Digest = (text: string) => Promise<string>;

export function createTaskRepository(db: SqlDriver, digest: Digest) {
  // All access to this private connection is serialized, including reads. This
  // keeps foreign_keys=ON and prevents outside queries joining an async transaction.
  let queue: Promise<unknown> = Promise.resolve();
  function serial<T>(work: () => Promise<T>): Promise<T> {
    const result = queue.then(work);
    queue = result.catch(() => undefined);
    return result;
  }
  async function transaction<T>(work: () => Promise<T>): Promise<T> {
    await db.execAsync('BEGIN IMMEDIATE');
    try {
      const result = await work();
      await db.execAsync('COMMIT');
      return result;
    } catch (error) {
      await db.execAsync('ROLLBACK');
      throw error;
    }
  }
  return {
    initialize: () => serial(async () => {
      await db.execAsync('PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;');
      const version = await db.getFirstAsync<{ user_version: number }>('PRAGMA user_version');
      if (version && version.user_version > 1) throw new Error('База создана более новой версией приложения. Обновите приложение, не удаляя его.');
      if (!version?.user_version) await transaction(() => db.execAsync(taskSchema));
    }),
    list: () => serial(() => db.getAllAsync<TaskSummary>(`
      SELECT id, revision, number, queue_position AS queuePosition,
        json_extract(content_json, '$.bendNumber') AS bendNumber,
        json_extract(content_json, '$.targetAngleDeg') AS targetAngleDeg,
        json_extract(content_json, '$.pipeSnapshot.factoryNumber') AS factoryNumber,
        json_extract(content_json, '$.machineSnapshot.model') AS machineModel
      FROM mobile_tasks ORDER BY queue_position, id
    `)),
    get: (id: string) => serial(async () => {
      const row = await db.getFirstAsync<{ content_json: string; imported_at: string }>(
        'SELECT content_json, imported_at FROM mobile_tasks WHERE id = ?', id.toLowerCase());
      return row ? { task: JSON.parse(row.content_json) as TransferredTask, importedAt: row.imported_at } : null;
    }),
    packageCount: () => serial(async () => (await db.getFirstAsync<{ count: number }>(
      'SELECT COUNT(*) AS count FROM imported_packages'))?.count ?? 0),
    importPackage: async (text: string, fileName: string): Promise<ImportResult> => {
      // Revalidate at confirmation; a preview object is not a trusted write model.
      const data = parseTaskPackage(text);
      const packageId = await digest(canonicalJson(data));
      const tasks = await Promise.all(data.tasks.map(async (task) => {
        const content = canonicalJson(task);
        return { task, content, hash: await digest(content) };
      }));
      return serial(() => transaction(async () => {
        const previous = await db.getFirstAsync<{ id: string }>('SELECT id FROM imported_packages WHERE id = ?', packageId);
        if (previous) return { added: 0, skipped: tasks.length, alreadyImported: true };
        const now = new Date().toISOString();
        await db.runAsync('INSERT INTO imported_packages(id, format, schema_version, file_name, task_count, imported_at) VALUES (?, ?, ?, ?, ?, ?)',
          packageId, data.format, data.schemaVersion, fileName.slice(0, 255), tasks.length, now);
        let added = 0;
        for (const { task, content, hash } of tasks) {
          const id = task.id.toLowerCase();
          const existing = await db.getFirstAsync<{ revision: number; content_sha256: string }>(
            'SELECT revision, content_sha256 FROM mobile_tasks WHERE id = ?', id);
          if (existing && (existing.revision !== task.revision || existing.content_sha256 !== hash)) {
            throw new Error(`Конфликт задания ${task.number}: этот UUID уже сохранён с другой ревизией или содержимым. Пакет не загружен. Проверьте его на ПК.`);
          }
          if (!existing) {
            const occupied = await db.getFirstAsync<{ number: string }>('SELECT number FROM mobile_tasks WHERE queue_position = ?', task.queuePosition);
            if (occupied) throw new Error(`Позиция ${task.queuePosition} уже занята заданием ${occupied.number}. Пакет не загружен. Очередь должна приходить с одного ПК.`);
            await db.runAsync('INSERT INTO mobile_tasks(id, revision, number, queue_position, content_json, content_sha256, first_package_id, imported_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)',
              id, task.revision, task.number, task.queuePosition, content, hash, packageId, now);
            added++;
          }
          await db.runAsync('INSERT INTO imported_package_tasks(package_id, task_id, task_revision) VALUES (?, ?, ?)', packageId, id, task.revision);
        }
        return { added, skipped: tasks.length - added, alreadyImported: false };
      }));
    },
  };
}
export type TaskRepository = ReturnType<typeof createTaskRepository>;
