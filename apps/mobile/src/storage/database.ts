import { openDatabaseAsync } from 'expo-sqlite';
import { CryptoDigestAlgorithm, digestStringAsync } from 'expo-crypto';
import { createTaskRepository, type TaskRepository } from './taskRepository';

let opening: Promise<TaskRepository> | undefined;
export function openTaskRepository(): Promise<TaskRepository> {
  opening ??= (async () => {
    const db = await openDatabaseAsync('pipe-bending-mobile.sqlite3', { useNewConnection: true });
    try {
      const repository = createTaskRepository(db, (text) => digestStringAsync(CryptoDigestAlgorithm.SHA256, text));
      await repository.initialize();
      return repository;
    } catch (error) { await db.closeAsync(); throw error; }
  })().catch((error: unknown) => { opening = undefined; throw error; });
  return opening;
}
