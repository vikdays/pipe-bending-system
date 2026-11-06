import { useEffect, useState } from 'react';
import { Modal } from '../../components/Modal';
import { authErrorMessage } from '../auth/appError';
import { ConfigurationFormModal } from './ConfigurationFormModal';
import type { Configuration, ConfigurationInput, Machine } from './taskTypes';
import type {
  TechnicalCardPreview,
  TechnicalCardService,
} from './technicalCardService';

export function TechnicalCardsModal({
  machines,
  configurations,
  service,
  onConfigurationSaved,
  onImported,
  onClose,
}: {
  machines: Machine[];
  configurations: Configuration[];
  service: TechnicalCardService;
  onConfigurationSaved: (
    input: ConfigurationInput,
    existing?: Configuration,
  ) => Promise<Configuration>;
  onImported: () => Promise<void>;
  onClose: () => void;
}) {
  const [machineId, setMachineId] = useState(machines[0]?.id ?? '');
  const [tab, setTab] = useState<'catalog' | 'import'>('catalog');
  const [history, setHistory] = useState<TechnicalCardPreview[]>([]);
  const [preview, setPreview] = useState<TechnicalCardPreview | null>(null);
  const [selection, setSelection] = useState<string[]>([]);
  const [query, setQuery] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [editor, setEditor] = useState<Configuration | null>(null);
  useEffect(() => {
    let active = true;
    service
      .history()
      .then((rows) => {
        if (active) setHistory(rows);
      })
      .catch((e: unknown) => {
        if (active) setError(authErrorMessage(e));
      });
    return () => {
      active = false;
    };
  }, [service]);
  async function run(operation: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await operation();
    } catch (e) {
      setError(authErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }
  function show(p: TechnicalCardPreview) {
    setPreview(p);
    setMachineId(p.machine.id);
    setSelection([]);
    setQuery('');
    setTab('import');
  }
  async function prepare() {
    await run(async () => {
      const p = await service.prepare(machineId);
      if (p) {
        show(p);
        setHistory(await service.history());
      }
    });
  }
  async function commit() {
    if (!preview) return;
    await run(async () => {
      const added = await service.commit(preview.id, selection);
      setPreview(await service.get(preview.id));
      setSelection([]);
      await onImported();
      setNotice(
        `Добавлено конфигураций: ${added}. Перед подготовкой заданий сверьте нужные варианты с оригиналом.`,
      );
    });
  }
  async function save(input: ConfigurationInput) {
    if (!editor || busy) return;
    setBusy(true);
    setError('');
    try {
      await onConfigurationSaved(input, editor);
      setEditor(null);
      setNotice('Конфигурация сохранена.');
    } catch (e) {
      setError(authErrorMessage(e));
      throw e;
    } finally {
      setBusy(false);
    }
  }
  const normalized = query.trim().toLocaleLowerCase('ru-RU');
  const configs = configurations.filter(
    (c) =>
      (!machineId || c.machineId === machineId) &&
      `${c.name} ${c.version}`.toLocaleLowerCase('ru-RU').includes(normalized),
  );
  const rows = (preview?.rows ?? []).filter((r) =>
    r.input.name.toLocaleLowerCase('ru-RU').includes(normalized),
  );
  return (
    <>
      <Modal
        extraWide
        title="Техкарты и конфигурации"
        description="Оригиналы хранятся на этом компьютере. Импорт поддерживает таблицы двух изученных DOCX для ГТ532 и ГТ1022."
        closeDisabled={busy || !!editor}
        onClose={onClose}
      >
        <div className="document-toolbar">
          <button
            className={`button ${tab === 'catalog' ? 'button--primary' : 'button--secondary'}`}
            disabled={busy}
            onClick={() => {
              setTab('catalog');
              setQuery('');
            }}
          >
            Конфигурации ({configurations.length})
          </button>
          <button
            className={`button ${tab === 'import' ? 'button--primary' : 'button--secondary'}`}
            disabled={busy}
            onClick={() => {
              setTab('import');
              setQuery('');
            }}
          >
            Импорт DOCX
          </button>
        </div>
        <div className="form-grid card-catalog-filters">
          <label className="form-field">
            Станок
            <select
              value={machineId}
              disabled={busy}
              onChange={(e) => {
                setMachineId(e.target.value);
                if (tab === 'import') {
                  setPreview(null);
                  setSelection([]);
                }
              }}
            >
              {machines.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
          </label>
          <label className="form-field">
            Поиск конфигурации
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Например: Ø530 или 9°"
            />
          </label>
        </div>
        {tab === 'catalog' ? (
          <>
            <p className="auth-help">
              Показано: {configs.length}. Отдельный вариант соответствует строке
              или столбцу таблицы ТК. Сверка подтверждает перенос данных;
              применимость оснастки и материалов проверяет руководитель.
            </p>
            <div className="card-catalog-list">
              {configs.map((c) => (
                <article className="card-catalog-row" key={c.id}>
                  <div>
                    <strong>{c.name}</strong>
                    <p>
                      {c.needsReview
                        ? 'Требуется сверка'
                        : 'Сверка не требуется или подтверждена'}{' '}
                      · {c.version}
                    </p>
                    <p>
                      Минимальная длина: {c.minPipeLengthMm ?? 'не указана'}
                      {c.minPipeLengthMm != null ? ' мм' : ''}
                    </p>
                  </div>
                  <button
                    className="button button--secondary"
                    disabled={busy}
                    onClick={() => {
                      setError('');
                      setEditor(c);
                    }}
                  >
                    Проверить
                  </button>
                </article>
              ))}
            </div>
            {!configs.length && (
              <p className="auth-help">
                Конфигураций пока нет. Перейдите в «Импорт DOCX» или создайте
                конфигурацию в форме задания.
              </p>
            )}
          </>
        ) : (
          <>
            <div className="document-toolbar">
              <button
                className="button button--primary"
                disabled={busy || !machineId}
                onClick={() => {
                  void prepare();
                }}
              >
                Выбрать техкарту DOCX
              </button>
            </div>
            <label className="form-field">
              Ранее загруженные карты
              <select
                value={preview?.id ?? ''}
                disabled={busy}
                onChange={(e) => {
                  const id = e.target.value;
                  if (id) void run(async () => show(await service.get(id)));
                  else {
                    setPreview(null);
                    setSelection([]);
                  }
                }}
              >
                <option value="">Выберите сохранённую карту</option>
                {history
                  .filter((p) => p.machine.id === machineId)
                  .map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.machine.name} · {p.originalName}
                    </option>
                  ))}
              </select>
            </label>
            {preview && (
              <>
                <div className="source-document-note">
                  <p>
                    <strong>{preview.machine.name}</strong> ·{' '}
                    {preview.originalName}
                  </p>
                  <p className="auth-help">
                    Вариантов: {preview.rows.length}. Уже добавлено:{' '}
                    {preview.rows.filter((r) => r.existingId).length}.
                  </p>
                  <button
                    className="button button--secondary"
                    disabled={busy}
                    onClick={() => {
                      void run(() => service.openOriginal(preview.id));
                    }}
                  >
                    Открыть оригинал DOCX
                  </button>
                </div>
                {preview.warnings.map((w) => (
                  <p key={w} className="auth-help">
                    {w}
                  </p>
                ))}
                <div className="document-toolbar">
                  <button
                    className="button button--secondary"
                    disabled={busy}
                    onClick={() =>
                      setSelection([
                        ...new Set([
                          ...selection,
                          ...rows
                            .filter((r) => !r.existingId)
                            .map((r) => r.key),
                        ]),
                      ])
                    }
                  >
                    Выбрать все показанные новые
                  </button>
                  <button
                    className="button button--secondary"
                    disabled={busy}
                    onClick={() => setSelection([])}
                  >
                    Снять выбор
                  </button>
                  <span>Выбрано: {selection.length}</span>
                </div>
                <div className="card-catalog-list">
                  {rows.map((row) => (
                    <label key={row.key} className="card-import-row">
                      <input
                        type="checkbox"
                        disabled={busy || !!row.existingId}
                        checked={selection.includes(row.key)}
                        onChange={(e) =>
                          setSelection((previous) =>
                            e.target.checked
                              ? [...previous, row.key]
                              : previous.filter((key) => key !== row.key),
                          )
                        }
                      />
                      <span>
                        <strong>{row.input.name}</strong>
                        <small>
                          {row.existingId
                            ? 'Уже добавлена'
                            : `Минимальная длина ${row.input.minPipeLengthMm ?? 'не распознана'} мм`}
                        </small>
                        {(row.warnings ?? []).map((w) => (
                          <small key={w}>{w}</small>
                        ))}
                      </span>
                    </label>
                  ))}
                </div>
                <div className="modal-actions">
                  <button
                    className="button button--primary"
                    disabled={busy || !selection.length}
                    onClick={() => {
                      void commit();
                    }}
                  >
                    Добавить конфигурации ({selection.length})
                  </button>
                </div>
              </>
            )}
          </>
        )}
        {busy && <p role="status">Обработка…</p>}
        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}
        {notice && (
          <p role="status" className="auth-help">
            {notice}
          </p>
        )}
        <div className="modal-actions">
          <button
            className="button button--secondary"
            disabled={busy}
            onClick={onClose}
          >
            Закрыть
          </button>
        </div>
      </Modal>
      {editor && (
        <ConfigurationFormModal
          key={editor.id}
          configuration={editor}
          machines={machines}
          busy={busy}
          error={error}
          onSave={save}
          onOpenOriginal={service.openOriginal}
          onClose={() => {
            if (!busy) {
              setEditor(null);
              setError('');
            }
          }}
        />
      )}
    </>
  );
}
