import { useCallback, useEffect, useRef, useState } from 'react';
import { useTaskRepository } from '@/storage/TaskStorageProvider';
import { pickTaskPackage, type PackagePreview } from './pickTaskPackage';
import type { TaskSummary } from './taskTypes';

export function useTaskList() {
  const repository = useTaskRepository();
  const [tasks, setTasks] = useState<TaskSummary[]>([]);
  const [packages, setPackages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [preview, setPreview] = useState<PackagePreview | null>(null);
  const importing = useRef(false);
  const active = useRef(true);
  const reload = useCallback(async () => {
    const [next, count] = await Promise.all([repository.list(), repository.packageCount()]);
    if (active.current) { setTasks(next); setPackages(count); }
  }, [repository]);
  useEffect(() => {
    active.current = true;
    reload().catch(() => { if (active.current) setError('Не удалось прочитать задания. Обновите список.'); })
      .finally(() => { if (active.current) setLoading(false); });
    return () => { active.current = false; };
  }, [reload]);
  async function choose() {
    if (importing.current) return;
    importing.current = true; setBusy(true); setError(''); setMessage('');
    try { const selected = await pickTaskPackage(); if (active.current) setPreview(selected); }
    catch (cause) { if (active.current) setError(cause instanceof Error ? cause.message : 'Не удалось прочитать файл.'); }
    finally { importing.current = false; if (active.current) setBusy(false); }
  }
  async function confirm() {
    if (!preview || importing.current) return;
    importing.current = true; setBusy(true); setError('');
    try {
      const result = await repository.importPackage(preview.text, preview.name);
      if (active.current) {
        setPreview(null);
        setMessage(result.alreadyImported ? 'Этот пакет уже загружен. Дубликаты не созданы.'
          : `Сохранено заданий: ${result.added}. Уже имелось: ${result.skipped}.`);
      }
      try { await reload(); }
      catch { if (active.current) setError('Пакет сохранён, но список не обновился. Потяните список вниз для обновления.'); }
    } catch (cause) {
      if (active.current) { setPreview(null); setError(cause instanceof Error ? cause.message : 'Не удалось сохранить пакет. Существующие задания не изменены.'); }
    } finally { importing.current = false; if (active.current) setBusy(false); }
  }
  async function refresh() {
    setLoading(true); setError('');
    try { await reload(); } catch { setError('Не удалось обновить список заданий.'); }
    finally { if (active.current) setLoading(false); }
  }
  return { tasks, packages, loading, busy, error, message, preview, choose, confirm, refresh, cancel: () => { if (!importing.current) setPreview(null); } };
}
