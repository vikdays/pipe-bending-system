import { Icon } from '../../components/Icon';
import { Modal } from '../../components/Modal';
import { TaskStatusBadge } from './TaskStatusBadge';
import { readAdditionalFields } from '../pipes/pipeFields';
import { PipeSourceDetails } from '../pipes/PipeSourceDetails';
import { TechnicalCardReferenceDetails } from './TechnicalCardReferenceDetails';
import { splitConfigurationParameters } from './configurationParameters';
import type { Task } from './taskTypes';

export function TaskDetailsModal({
  task,
  onClose,
  busy,
  error,
  onEdit,
  onPrepare,
  onReviewConfiguration,
  onReload,
  onOpenOriginal,
}: {
  task: Task;
  onClose: () => void;
  busy: boolean;
  error: string;
  onEdit: (task: Task) => void;
  onPrepare: (task: Task) => void;
  onReviewConfiguration: (task: Task) => void;
  onReload: (task: Task) => void;
  onOpenOriginal: (id: string) => Promise<void>;
}) {
  const fields = [
    ['Порядок в очереди', String(task.queuePosition)],
    ['Ревизия задания', String(task.revision)],
    ['Планируемый отвод', task.bendNumber],
    ['Тип отвода', task.bendType || 'Не выбран'],
    ['Станок', task.machineSnapshot?.name || task.machine || 'Не выбран'],
    [
      'Заводской номер станка',
      task.machineSnapshot?.serialNumber || 'Не указан',
    ],
    [
      'Погода и температура',
      'Фактические условия вводятся при начале сеанса на планшете. Экран сеанса ещё не реализован.',
    ],
    ['Целевой угол', task.targetAngle ? `${task.targetAngle}°` : 'Не указан'],
    ['Дата создания', task.date],
    [
      'Технологическая конфигурация',
      task.configurationSnapshot
        ? `${task.configurationSnapshot.name} · версия ${task.configurationSnapshot.version} · ревизия ${task.configurationRevision}`
        : 'Не выбрана',
    ],
    [
      'Технологическая карта',
      task.configurationSnapshot?.technicalCardReference || 'Не указана',
    ],
    [
      'Подготовлено к передаче',
      task.preparedAt
        ? new Date(task.preparedAt).toLocaleString('ru-RU')
        : 'Пока не подготовлено',
    ],
    [
      'Успешная выгрузка',
      task.exportedAt
        ? new Date(task.exportedAt).toLocaleString('ru-RU')
        : task.status === 'exported'
          ? 'Записана в журнале обмена'
          : 'Пока не выгружено',
    ],
    ['Примечание', task.notes || '—'],
  ];

  const pipe = task.pipeSnapshot;
  const pipeFields = [
    ['Инвентарный номер записи', pipe.inventoryNumber],
    ['Заводской номер трубы', pipe.factoryNumber],
    [
      'Наружный диаметр',
      pipe.diameterMm == null ? 'Не указан в журнале' : `${pipe.diameterMm} мм`,
    ],
    [
      'Толщина стенки',
      pipe.wallThicknessMm == null
        ? 'Не указана в журнале'
        : `${pipe.wallThicknessMm} мм`,
    ],
    [
      'Длина трубы',
      pipe.lengthMm == null ? 'Не указана в журнале' : `${pipe.lengthMm} мм`,
    ],
    ['Класс прочности', pipe.strengthClass || 'Не указан в журнале'],
    ['Марка стали', pipe.steelGrade || 'Не указана в журнале'],
    ['Изготовитель', pipe.manufacturer || 'Не указан в журнале'],
    ['Номер плавки', pipe.heatNumber || 'Не указан в журнале'],
    [
      'Сертификаты',
      pipe.certificates.map((c) => c.reference).join('; ') ||
        'Не указаны в журнале',
    ],
  ];
  let additionalFields: ReturnType<typeof readAdditionalFields> = [];
  try {
    additionalFields = readAdditionalFields(pipe.attributesJson);
  } catch {
    /* Missing legacy metadata does not hide the pipe itself. */
  }
  const modeParameters = splitConfigurationParameters(
    task.configurationSnapshot?.parameters ?? [],
  ).mode;

  return (
    <Modal
      wide
      title={`Задание ${task.number}`}
      description={
        task.status === 'draft'
          ? 'Данные сохранённого черновика. Перед подготовкой проверьте трубу, конфигурацию и шапку будущего протокола.'
          : task.status === 'exported'
            ? 'Задание выгружено в JSON-файл. Получение планшетом пока не подтверждено. Снимки и ревизия сохранены.'
            : 'Задание подготовлено. Ревизия и снимки исходных данных зафиксированы; изменение запрещено.'
      }
      closeDisabled={busy}
      onClose={onClose}
    >
      <div className="task-detail-status">
        <TaskStatusBadge status={task.status} />
      </div>
      <dl className="detail-grid">
        {fields.map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      <section className="task-data-section">
        <h3>Конкретная труба</h3>
        <p className="auth-help">
          Источник: запись трубного журнала, версия {task.pipeRevision}. Здесь
          показаны сохранённые в задании сведения о трубе {pipe.factoryNumber}.
        </p>
        <dl className="detail-grid">
          {pipeFields.map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
        {!!additionalFields.length && (
          <details className="pipe-additional-details">
            <summary>Дополнительные сведения о трубе из журнала</summary>
            <dl className="detail-grid">
              {additionalFields.map((p, i) => (
                <div key={`${p.name}-${i}`}>
                  <dt>{p.name}</dt>
                  <dd>
                    {p.value}
                    {p.unit ? ` ${p.unit}` : ''}
                  </dd>
                </div>
              ))}
            </dl>
          </details>
        )}
        <PipeSourceDetails attributesJson={pipe.attributesJson} />
      </section>
      {task.configurationSnapshot && (
        <section className="task-configuration-snapshot">
          <h3>Параметры выбранного режима</h3>
          <p className="auth-help">
            Значения выбранного табличного варианта из ТК. Исходные единицы и
            обозначения сносок сохранены.
          </p>
          {task.configurationSnapshot.needsReview && (
            <div className="source-document-note">
              <p className="form-error">
                В черновике сохранена конфигурация без подтверждённой сверки с
                ТК.
              </p>
              {task.status === 'draft' && (
                <p>
                  Нажмите «Проверить конфигурацию ТК» внизу карточки. Откройте
                  оригинал, сверьте параметры и сноски, отметьте подтверждение
                  сверки и сохраните его. Затем сохраните черновик, чтобы
                  обновить в нём версию конфигурации.
                </p>
              )}
            </div>
          )}
          {modeParameters.length ? (
            <dl className="detail-grid">
              {modeParameters.map((p) => (
                <div key={p.name}>
                  <dt>{p.name}</dt>
                  <dd>
                    {p.value}
                    {p.unit ? ` ${p.unit}` : ''}
                  </dd>
                </div>
              ))}
            </dl>
          ) : (
            <p className="auth-help">Параметры режима из ТК не внесены.</p>
          )}
          {task.configurationSnapshot.notes && (
            <p className="auth-help">{task.configurationSnapshot.notes}</p>
          )}
        </section>
      )}
      {task.configurationSnapshot && (
        <TechnicalCardReferenceDetails
          configuration={task.configurationSnapshot}
          onOpenOriginal={onOpenOriginal}
        />
      )}
      <div className="inline-note">
        <Icon name="info" size={18} />
        <p>
          {task.status === 'draft'
            ? 'Подготовка проверит обязательные поля и версии трубы и конфигурации, сохранит их снимки и заблокирует редактирование задания.'
            : task.status === 'exported'
              ? 'Задание уже выгружено и исключено из очереди следующей выгрузки. Сведения о файле и составе пакета доступны в журнале обмена.'
              : 'Изменения трубы или конфигурации в каталоге не меняют это задание. На главном экране нажмите «Выгрузить задания», чтобы сохранить все подготовленные задания одним файлом. Сохранение файла не подтверждает его получение машинистом.'}{' '}
          Автоматические производственные расчёты не выполняются; параметры
          необходимо сверить с ТК.
        </p>
      </div>
      {busy && <p role="status">Выполнение операции…</p>}
      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}
      <div className="modal-actions">
        <button
          className="button button--secondary"
          disabled={busy}
          onClick={() => onReload(task)}
        >
          Обновить карточку
        </button>
        {task.status === 'draft' && (
          <>
            <button
              className="button button--secondary"
              disabled={busy}
              onClick={() => onEdit(task)}
            >
              Изменить черновик
            </button>
            <button
              className="button button--primary"
              disabled={busy}
              onClick={() =>
                task.configurationSnapshot?.needsReview
                  ? onReviewConfiguration(task)
                  : onPrepare(task)
              }
            >
              {task.configurationSnapshot?.needsReview
                ? 'Проверить конфигурацию ТК'
                : 'Подготовить к передаче'}
            </button>
          </>
        )}
        <button
          className="button button--secondary"
          disabled={busy}
          onClick={onClose}
        >
          Закрыть
        </button>
      </div>
    </Modal>
  );
}
