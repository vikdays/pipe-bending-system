import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from 'react-native';
import { openTaskRepository } from './database';
import type { TaskRepository } from './taskRepository';

const StorageContext = createContext<TaskRepository | null>(null);
export function TaskStorageProvider({ children }: { children: ReactNode }) {
  const [repository, setRepository] = useState<TaskRepository | null>(null);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    setError('');
    openTaskRepository().then((value) => { if (active) setRepository(value); })
      .catch((cause: unknown) => { if (active) setError(cause instanceof Error ? cause.message : 'Не удалось открыть локальную базу.'); });
    return () => { active = false; };
  }, [attempt]);
  if (repository) return <StorageContext.Provider value={repository}>{children}</StorageContext.Provider>;
  return <View style={styles.container}>
    {error ? <><Text accessibilityRole="alert" style={styles.message}>{error}</Text>
      <Pressable accessibilityRole="button" onPress={() => setAttempt((n) => n + 1)} style={styles.button}><Text>Повторить</Text></Pressable></>
      : <><ActivityIndicator size="large" /><Text style={styles.message}>Открываем локальную базу…</Text></>}
  </View>;
}
export function useTaskRepository(): TaskRepository {
  const repository = useContext(StorageContext);
  if (!repository) throw new Error('Хранилище ещё не открыто.');
  return repository;
}
const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24, gap: 20, backgroundColor: '#f5f6f8' },
  message: { fontSize: 18, color: '#28374b', textAlign: 'center' },
  button: { padding: 18, borderWidth: 1, borderColor: '#9aa8bc', borderRadius: 8 },
});
