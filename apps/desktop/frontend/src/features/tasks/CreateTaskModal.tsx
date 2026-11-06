import { useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { Modal } from '../../components/Modal';
import { authErrorMessage } from '../auth/appError';
import { technicalCardService } from './technicalCardClient';
import { TaskDraftFields } from './TaskDraftFields';
import { ConfigurationFormModal } from './ConfigurationFormModal';
import { emptyTaskDraft } from './taskTypes';
import type {
  Configuration,
  ConfigurationInput,
  Machine,
  Task,
  TaskDraft,
} from './taskTypes';
import type { Pipe } from '../pipes/pipeTypes';

export function CreateTaskModal({
  task,
  reviewConfiguration = false,
  pipeOptions,
  machines,
  configurations,
  isSubmitting,
  error,
  onSave,
  onConfigurationSaved,
  onClose,
}: {
  task?: Task;
  reviewConfiguration?: boolean;
  pipeOptions: Pipe[];
  machines: Machine[];
  configurations: Configuration[];
  isSubmitting: boolean;
  error: string;
  onSave: (requestId: string, drafts: TaskDraft[]) => Promise<void>;
  onConfigurationSaved: (
    input: ConfigurationInput,
    existing?: Configuration,
  ) => Promise<Configuration>;
  onClose: () => void;
}) {
  const [rows, setRows] = useState(() => [
    {
      key: crypto.randomUUID(),
      input: task
        ? {
            number: task.number,
            pipeId: task.pipeId,
            pipeRevision: task.pipeRevision,
            bendNumber: task.bendNumber,
            machine: task.machine,
            machineId: task.machineId,
            targetAngle: task.targetAngle,
            bendType: task.bendType,
            notes: task.notes,
            configurationId: task.configurationId,
            configurationRevision: task.configurationRevision,
          }
        : emptyTaskDraft(),
    },
  ]);
  const request = useRef({ signature: '', id: crypto.randomUUID() });
  const [editor, setEditor] = useState<{
    row: string;
    configuration?: Configuration;
  } | null>(() => {
    const configuration = configurations.find(
      (c) => c.id === task?.configurationId,
    );
    return reviewConfiguration && configuration?.needsReview
      ? { row: rows[0].key, configuration }
      : null;
  });
  const [configurationBusy, setConfigurationBusy] = useState(false);
  const [configurationError, setConfigurationError] = useState('');
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (isSubmitting) return;
    const inputs = rows.map(({ input }) => {
      const pipe = pipeOptions.find((p) => p.id === input.pipeId);
      const c = configurations.find((c) => c.id === input.configurationId);
      return {
        ...input, //spread-синтаксис (копирует все текущие поля формы в новый объект)
        pipeRevision: pipe?.revision ?? 0,
        configurationRevision: c?.revision ?? 0,
        machine: c?.machine ?? input.machine,
      };
    });
    const signature = JSON.stringify(inputs);
    if (request.current.signature !== signature)
      request.current = { signature, id: crypto.randomUUID() };
    void onSave(request.current.id, inputs);
  }
  function move(index: number, offset: number) {
    setRows((previous) => {
      const next = [...previous]; //копия массива
      [next[index], next[index + offset]] = [next[index + offset], next[index]];
      return next;
    });
  }
  async function saveConfiguration(input: ConfigurationInput) {
    if (!editor || configurationBusy) return;
    setConfigurationBusy(true);
    setConfigurationError('');
    try {
      const c = await onConfigurationSaved(input, editor.configuration);
      setRows((previous) =>
        previous.map((row) =>
          row.key === editor.row
            ? {
                ...row,
                input: {
                  ...row.input,
                  configurationId: c.id,
                  configurationRevision: c.revision,
                  machine: c.machine,
                  machineId: c.machineId,
                },
              }
            : row,
        ),
      );
      setEditor(null);
    } catch (e) {
      setConfigurationError(authErrorMessage(e));
    } finally {
      setConfigurationBusy(false);
    }
  }
  return (
    <>
      <Modal
        extraWide
        title={task ? `Изменить черновик ${task.number}` : 'Новые задания'}
        description={
          task
            ? 'При сохранении черновика сведения о трубе и конфигурации обновятся до показанных версий. Порядок в очереди сохранится.'
            : 'Одна карточка — один отвод из конкретной трубы. Задания попадут в конец очереди в указанном порядке.'
        }
        closeDisabled={isSubmitting || configurationBusy}
        onClose={onClose}
      >
        <form onSubmit={submit} className="task-form">
          {reviewConfiguration && (
            <div className="source-document-note" role="status">
              {configurations.some(
                (c) =>
                  c.id === rows[0].input.configurationId &&
                  c.technicalCardId &&
                  !c.needsReview,
              )
                ? 'Сверка выбранной конфигурации подтверждена. Проверьте данные ниже и нажмите «Сохранить черновик»: в задание будет записана актуальная версия конфигурации. Затем откройте карточку и подготовьте задание к передаче.'
                : 'Сначала проверьте конфигурацию по оригиналу ТК и подтвердите сверку. После этого сохраните черновик, чтобы обновить его данные.'}
            </div>
          )}
          <fieldset
            className="pipe-form-fields"
            disabled={isSubmitting || configurationBusy}
          >
            {rows.map((row, index) => (
              <section key={row.key} className="task-draft-card">
                <div className="task-draft-heading">
                  <h3>{task ? 'Данные черновика' : `Задание ${index + 1}`}</h3>
                  {!task && (
                    <div className="document-toolbar">
                      <button
                        type="button"
                        className="button button--secondary"
                        disabled={index === 0}
                        aria-label={`Переместить задание ${index + 1} выше`}
                        onClick={() => move(index, -1)}
                      >
                        Выше
                      </button>
                      <button
                        type="button"
                        className="button button--secondary"
                        disabled={index === rows.length - 1}
                        aria-label={`Переместить задание ${index + 1} ниже`}
                        onClick={() => move(index, 1)}
                      >
                        Ниже
                      </button>
                      <button
                        type="button"
                        className="button button--secondary"
                        disabled={rows.length === 1}
                        aria-label={`Убрать задание ${index + 1}`}
                        onClick={() =>
                          setRows((previous) =>
                            previous.filter((r) => r.key !== row.key),
                          )
                        }
                      >
                        Убрать
                      </button>
                    </div>
                  )}
                </div>
                <TaskDraftFields
                  value={row.input}
                  pipes={pipeOptions}
                  machines={machines}
                  configurations={configurations}
                  onChange={(input) =>
                    setRows((previous) =>
                      previous.map((r) =>
                        r.key === row.key ? { ...r, input } : r,
                      ),
                    )
                  }
                  onConfiguration={(configuration) => {
                    setConfigurationError('');
                    setEditor({ row: row.key, configuration });
                  }}
                />
              </section>
            ))}
            {!task && (
              <button
                type="button"
                className="button button--secondary"
                disabled={rows.length >= 50}
                onClick={() =>
                  setRows((previous) => {
                    const last = previous.at(-1)!.input;
                    return [
                      ...previous,
                      {
                        key: crypto.randomUUID(),
                        input: {
                          ...emptyTaskDraft(),
                          configurationId: last.configurationId,
                          configurationRevision: last.configurationRevision,
                          machine: last.machine,
                          machineId: last.machineId,
                          bendType: last.bendType,
                          targetAngle: last.targetAngle,
                        },
                      },
                    ];
                  })
                }
              >
                Добавить ещё задание
              </button>
            )}
            <p className="auth-help">
              Для черновика достаточно выбрать трубу. Перед подготовкой
              понадобятся номер и тип отвода, положительный целевой угол,
              конфигурация и заполненные размеры трубы. Погода и температура
              будут вводиться при начале сеанса на планшете. Измерения
              записываются во время гибки после подтверждения машинистом. Экран
              сеанса ещё не реализован; в задании текущую погоду вводить не
              нужно.
            </p>
          </fieldset>
          {error && (
            <p role="alert" className="form-error">
              {error}
            </p>
          )}
          <div className="modal-actions">
            <button
              type="button"
              className="button button--secondary"
              disabled={isSubmitting || configurationBusy}
              onClick={onClose}
            >
              Отмена
            </button>
            <button
              className="button button--primary"
              disabled={
                isSubmitting || configurationBusy || pipeOptions.length === 0
              }
            >
              {isSubmitting
                ? 'Сохранение…'
                : task
                  ? 'Сохранить черновик'
                  : `Создать черновики (${rows.length})`}
            </button>
          </div>
        </form>
      </Modal>
      {editor && (
        <ConfigurationFormModal
          configuration={editor.configuration}
          machines={machines}
          onOpenOriginal={technicalCardService.openOriginal}
          busy={configurationBusy}
          error={configurationError}
          onSave={saveConfiguration}
          requireSourceReview={reviewConfiguration}
          onClose={() => setEditor(null)}
        />
      )}
    </>
  );
}
