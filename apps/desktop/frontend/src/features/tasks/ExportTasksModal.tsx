import { useEffect, useState } from 'react';
import { Modal } from '../../components/Modal';
import { authErrorMessage } from '../auth/appError';
import type { TaskService, TaskExportResult } from './taskService';
import type { Task } from './taskTypes';

export function ExportTasksModal({
  service,
  onSaved,
  onClose,
}: {
  service: TaskService;
  onSaved: (result: TaskExportResult) => void;
  onClose: () => void;
}) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    service
      .getTasks()
      .then((rows) => {
        if (!active) return;
        setTasks(
          rows
            .filter((t) => t.status === 'ready')
            .sort((a, b) => a.queuePosition - b.queuePosition),
        );
        setLoading(false);
      })
      .catch((e: unknown) => {
        if (active) {
          setError(authErrorMessage(e));
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [service, attempt]);
  async function save() {
    if (busy || loading || !tasks.length) return;
    setBusy(true);
    setError('');
    setNotice('');
    try {
      const result = await service.exportTasks(
        tasks.map((t) => ({ id: t.id, revision: t.revision })),
      );
      if (result.outcome === 'saved') onSaved(result);
      else if (result.outcome === 'written_unlogged') setError(result.message);
      else setNotice('Сохранение отменено. Файл не выгружен.');
    } catch (e) {
      setError(authErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      wide
      title="Выгрузить задания"
      description="Все подготовленные и ещё не выгруженные задания сохранятся одним файлом в указанном порядке."
      closeDisabled={busy}
      onClose={onClose}
    >
      <p className="auth-help">
        Черновики, уже выгруженные и завершённые задания не входят в пакет.
        Поиск и фильтры главного экрана не ограничивают состав выгрузки.
      </p>
      <button
        className="button button--secondary"
        disabled={busy || loading}
        onClick={() => {
          setLoading(true);
          setError('');
          setNotice('');
          setAttempt((n) => n + 1);
        }}
      >
        Обновить состав пакета
      </button>
      {loading ? (
        <p role="status">Загрузка заданий…</p>
      ) : tasks.length ? (
        <>
          <p>
            Заданий в одном файле: <strong>{tasks.length}</strong>
          </p>
          <ol className="export-task-list">
            {tasks.map((task) => (
              <li key={task.id}>
                <strong>{task.number}</strong> · отвод {task.bendNumber} ·
                ревизия {task.revision}
                <br />
                Очередь: {task.queuePosition} · труба {task.pipeNumber}
              </li>
            ))}
          </ol>
        </>
      ) : (
        !error && (
          <p>
            Заданий, ожидающих выгрузки, нет. Подготовьте новые задания к
            передаче в их карточках.
          </p>
        )
      )}
      {busy && <p role="status">Сохранение пакета…</p>}
      {notice && <p role="status">{notice}</p>}
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
      <div className="modal-actions">
        <button
          className="button button--primary"
          disabled={loading || busy || !tasks.length || tasks.length > 1000}
          onClick={() => {
            void save();
          }}
        >
          Сохранить пакет ({tasks.length})
        </button>
        <button
          className="button button--secondary"
          disabled={busy}
          onClick={onClose}
        >
          Закрыть
        </button>
      </div>
      {tasks.length > 1000 && (
        <p className="form-error">
          В одном пакете поддерживается до 1000 заданий.
        </p>
      )}
    </Modal>
  );
}
