import { useEffect, useState } from 'react';
import { Modal } from '../../components/Modal';
import { authErrorMessage } from '../auth/appError';
import type { TaskService } from '../tasks/taskService';
import type { ExchangeEvent } from '../../../bindings/changeme/internal/registry/models';

export function ExchangeInfoModal({
  service,
  onClose,
}: {
  service: TaskService;
  onClose: () => void;
}) {
  const [events, setEvents] = useState<ExchangeEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    service
      .getExchangeEvents()
      .then((rows) => {
        if (active) {
          setEvents(rows);
          setLoading(false);
        }
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
  return (
    <Modal
      wide
      title="Журнал обмена"
      description="Последние 100 операций выгрузки на этом компьютере."
      onClose={onClose}
    >
      <p className="auth-help">
        «Файл сохранён» означает успешную запись JSON. Получение и импорт на
        планшете пока не подтверждаются. Отмена диалога не создаёт запись.
      </p>
      <button
        className="button button--secondary"
        disabled={loading}
        onClick={() => {
          setLoading(true);
          setError('');
          setAttempt((n) => n + 1);
        }}
      >
        Обновить журнал
      </button>
      {loading ? (
        <p role="status">Загрузка журнала…</p>
      ) : error ? (
        <p role="alert" className="form-error">
          {error}
        </p>
      ) : events.length === 0 ? (
        <p className="auth-help">
          Выгрузок пока нет. Подготовьте задания и нажмите «Выгрузить задания»
          на главном экране.
        </p>
      ) : (
        <div className="exchange-events">
          {events.map((event) => (
            <article key={event.id} className="exchange-event">
              <strong>Пакет заданий: {event.tasks?.length ?? 0}</strong>
              <p>
                {event.status === 'saved'
                  ? 'Файл сохранён'
                  : event.status === 'failed'
                    ? 'Ошибка записи'
                    : 'Результат не подтверждён'}
              </p>
              <p>
                {event.fileName} ·{' '}
                {new Date(event.createdAt).toLocaleString('ru-RU')}
              </p>
              {event.status === 'pending' && (
                <p className="auth-help">
                  Операция выполняется или была прервана. Проверьте наличие
                  файла и при необходимости повторите выгрузку.
                </p>
              )}
              {event.errorMessage && (
                <p className="form-error">{event.errorMessage}</p>
              )}
              <details>
                <summary>UUID и контрольная сумма файла</summary>
                <ol>
                  {(event.tasks ?? []).map((task) => (
                    <li key={task.taskId}>
                      {task.taskNumber} · ревизия {task.taskRevision} · очередь{' '}
                      {task.queuePosition}
                      <br />
                      UUID: {task.taskUuid}
                    </li>
                  ))}
                </ol>
                <p>SHA-256: {event.sha256}</p>
                <p>Размер: {event.sizeBytes} байт</p>
              </details>
            </article>
          ))}
        </div>
      )}
      <div className="modal-actions">
        <button className="button button--secondary" onClick={onClose}>
          Закрыть
        </button>
      </div>
    </Modal>
  );
}
