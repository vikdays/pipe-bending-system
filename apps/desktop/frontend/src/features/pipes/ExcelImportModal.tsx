import { useState } from 'react';
import { Modal } from '../../components/Modal';
import { PipeFormModal } from './PipeFormModal';
import { ExcelMapping } from './ExcelMapping';
import { useExcelImport } from './useExcelImport';
import type { Certificate } from './pipeTypes';
import type { DocumentService } from './documentService';

const statuses: Record<string, string> = {
  new: 'Новая',
  existing: 'Уже в журнале',
  conflict: 'Конфликт',
  invalid: 'Ошибка',
  duplicate: 'Повтор в файле',
  imported: 'Добавлена',
};
export function ExcelImportModal({
  service,
  certificates,
  onImported,
  onClose,
}: {
  service: DocumentService;
  certificates: Certificate[];
  onImported: () => void;
  onClose: () => void;
}) {
  const model = useExcelImport(service, onImported);
  const [editing, setEditing] = useState('');
  const [filter, setFilter] = useState('all');
  const [page, setPage] = useState(0);
  const p = model.preview;
  const row = p?.rows.find((item) => item.id === editing);
  const filtered =
    p?.rows.filter((item) => filter === 'all' || item.status === filter) ?? [];
  const pages = Math.max(1, Math.ceil(filtered.length / 50));
  const current = Math.min(page, pages - 1);
  const committed = p?.status === 'committed';
  return (
    <>
      <Modal
        extraWide
        title="Импорт труб из Excel"
        description="Оригинал и черновик сохраняются на этом компьютере. Добавление в журнал требует отдельного подтверждения."
        closeDisabled={model.busy}
        onClose={onClose}
      >
        <div className="document-toolbar">
          <button
            className="button button--primary"
            disabled={model.busy}
            onClick={model.prepare}
          >
            Выбрать .xlsx
          </button>
          <button
            className="button button--secondary"
            disabled={model.busy}
            onClick={model.refresh}
          >
            Обновить {p ? 'предпросмотр' : 'историю'}
          </button>
          {p && (
            <button
              className="button button--secondary"
              disabled={model.busy}
              onClick={model.back}
            >
              История импорта
            </button>
          )}
        </div>
        {model.busy && <p role="status">Обработка документа…</p>}
        {model.error && (
          <p className="form-error" role="alert">
            {model.error}
          </p>
        )}
        {!p ? (
          <div className="import-history">
            <p className="auth-help">
              Выберите книгу, затем проверьте лист «Трубный журнал»,
              соответствие столбцов и единицы. Пустые характеристики можно
              заполнить позже. Конфликты не перезаписывают существующие трубы.
            </p>
            <h3>Сохранённые импорты</h3>
            {!model.busy && model.history.length === 0 && (
              <p>Импортов пока нет.</p>
            )}
            {model.history.map((item) => (
              <button
                disabled={model.busy}
                className="import-history-item"
                key={item.id}
                onClick={() => {
                  setPage(0);
                  setFilter('all');
                  void model.resume(item.id);
                }}
              >
                <strong>{item.originalName}</strong>
                <span>
                  {item.sheet} ·{' '}
                  {item.status === 'committed'
                    ? `Завершён · добавлено ${item.result?.added ?? 0}`
                    : 'Черновик'}{' '}
                  · {new Date(item.createdAt).toLocaleString('ru-RU')}
                </span>
              </button>
            ))}
          </div>
        ) : (
          <>
            <div className="import-source">
              <strong>{p.originalName}</strong>
              <span>
                {p.configuration.sheet} · {p.rows.length} строк · версия
                предпросмотра {p.revision}
              </span>
              <button
                className="button button--secondary"
                disabled={model.busy}
                onClick={model.openOriginal}
              >
                Открыть оригинал
              </button>
            </div>
            {committed ? (
              <p className="import-result" role="status">
                Импорт завершён. Добавлено: {p.result?.added ?? 0}; уже в
                журнале: {p.result?.existing ?? 0}; не импортировано:{' '}
                {p.result?.notImported ?? 0}. Существующие записи не изменены.
              </p>
            ) : (
              <>
                <ExcelMapping
                  key={`${p.id}:${p.revision}`}
                  preview={p}
                  busy={model.busy}
                  onApply={model.configure}
                />
                <details className="pipe-form-section">
                  <summary>Примечания к чтению ({p.warnings.length})</summary>
                  {p.warnings.map((warning, index) => (
                    <p className="auth-help" key={index}>
                      {warning}
                    </p>
                  ))}
                </details>
              </>
            )}
            <div className="document-toolbar">
              <label className="form-field">
                Строки
                <select
                  value={filter}
                  onChange={(e) => {
                    setFilter(e.target.value);
                    setPage(0);
                  }}
                >
                  <option value="all">Все ({p.rows.length})</option>
                  {Object.entries(statuses).map(([key, label]) => (
                    <option key={key} value={key}>
                      {label} (
                      {p.rows.filter((item) => item.status === key).length})
                    </option>
                  ))}
                </select>
              </label>
              {!committed && (
                <>
                  <button
                    className="button button--secondary"
                    disabled={model.busy}
                    onClick={() =>
                      model.setSelected(
                        p.rows
                          .filter((item) => item.status === 'new')
                          .map((item) => item.id),
                      )
                    }
                  >
                    Выбрать все новые
                  </button>
                  <button
                    className="button button--secondary"
                    disabled={model.busy}
                    onClick={() => model.setSelected([])}
                  >
                    Снять выбор
                  </button>
                  <span>Выбрано: {model.selected.length}</span>
                </>
              )}
            </div>
            <div className="table-scroll import-table">
              <table className="task-table">
                <thead>
                  <tr>
                    <th scope="col">Выбор</th>
                    <th scope="col">Строка / состояние</th>
                    <th scope="col">Труба</th>
                    <th scope="col">Характеристики</th>
                    <th scope="col">Проверка</th>
                    <th scope="col">Действие</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered
                    .slice(current * 50, (current + 1) * 50)
                    .map((item) => (
                      <tr key={item.id}>
                        <td>
                          <input
                            type="checkbox"
                            aria-label={`Добавить строку ${item.sourceRow}`}
                            disabled={
                              model.busy || committed || item.status !== 'new'
                            }
                            checked={model.selected.includes(item.id)}
                            onChange={(e) =>
                              model.setSelected((previous) =>
                                e.target.checked
                                  ? [...previous, item.id]
                                  : previous.filter((id) => id !== item.id),
                              )
                            }
                          />
                        </td>
                        <td>
                          <strong>{item.sourceRow}</strong>
                          <span
                            className={`import-status import-status--${item.status}`}
                          >
                            {statuses[item.status] ?? item.status}
                          </span>
                        </td>
                        <td>
                          <strong>
                            {item.input.factoryNumber || 'Номер отсутствует'}
                          </strong>
                          <small className="pipe-cell-secondary">
                            {item.input.manufacturer ||
                              'Изготовитель не указан'}{' '}
                            · плавка {item.input.heatNumber || '—'}
                          </small>
                          <small className="pipe-cell-secondary">
                            Инв. №{' '}
                            {item.existingInventoryNumber ||
                              item.input.inventoryNumber ||
                              '—'}
                          </small>
                        </td>
                        <td>
                          {item.input.diameterMm ?? '—'} ×{' '}
                          {item.input.wallThicknessMm ?? '—'} мм
                          <small className="pipe-cell-secondary">
                            Длина {item.input.lengthMm ?? '—'} мм
                          </small>
                        </td>
                        <td>
                          {(item.issues ?? []).map((issue, index) => (
                            <small
                              className={
                                issue.severity === 'error'
                                  ? 'form-error pipe-cell-secondary'
                                  : 'pipe-cell-secondary'
                              }
                              key={index}
                            >
                              {issue.message}
                            </small>
                          ))}
                          {item.input.newCertificate && (
                            <small className="pipe-cell-secondary">
                              {item.input.newCertificate.reference}
                            </small>
                          )}
                        </td>
                        <td>
                          {!committed && (
                            <button
                              className="button button--secondary"
                              disabled={model.busy}
                              onClick={() => setEditing(item.id)}
                            >
                              Исправить
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
            {filtered.length === 0 && (
              <p>
                Нет строк с таким состоянием. Проверьте лист и строку
                заголовков, если список пуст.
              </p>
            )}
            <div className="document-toolbar">
              <button
                className="button button--secondary"
                disabled={current === 0}
                onClick={() => setPage(current - 1)}
              >
                Назад
              </button>
              <span>
                {current + 1} / {pages}
              </span>
              <button
                className="button button--secondary"
                disabled={current + 1 === pages}
                onClick={() => setPage(current + 1)}
              >
                Далее
              </button>
            </div>
            {!committed && (
              <div className="import-confirm">
                <p>
                  В журнал будут добавлены только выбранные новые строки.
                  Повторы и конфликты пропускаются. Все выбранные строки
                  сохраняются одной операцией.
                </p>
                <button
                  className="button button--primary"
                  disabled={model.busy || model.selected.length === 0}
                  onClick={model.commit}
                >
                  Добавить в журнал ({model.selected.length})
                </button>
              </div>
            )}
          </>
        )}
        <div className="modal-actions">
          <button
            className="button button--secondary"
            disabled={model.busy}
            onClick={onClose}
          >
            Закрыть
          </button>
        </div>
      </Modal>
      {row && (
        <PipeFormModal
          key={row.id}
          draft={row.input}
          certificates={certificates}
          busy={model.busy}
          error={model.error}
          onClose={() => setEditing('')}
          onSave={async (input) => {
            if (await model.edit(row.id, input)) setEditing('');
          }}
        />
      )}
    </>
  );
}
