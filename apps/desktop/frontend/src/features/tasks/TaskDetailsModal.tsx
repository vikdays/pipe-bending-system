import { Icon } from '../../components/Icon';
import { Modal } from '../../components/Modal';
import { TaskStatusBadge } from './TaskStatusBadge';
import type { Task } from './taskTypes';

export function TaskDetailsModal({
  task,
  onClose,
}: {
  task: Task;
  onClose: () => void;
}) {
  const fields = [
    ['Инвентарный номер записи', task.pipeInventoryNumber],
    ['Исходная труба', task.pipeNumber],
    ['Размеры трубы', task.dimensions],
    ['Длина трубы', `${task.pipeSnapshot.lengthMm ?? '—'} мм`],
    ['Изготовитель', task.pipeSnapshot.manufacturer || 'Не указан'],
    ['Плавка', task.pipeSnapshot.heatNumber || 'Не указана'],
    [
      'Сертификаты',
      task.pipeSnapshot.certificates
        .map((certificate) => certificate.reference)
        .join('; ') || 'Не указаны',
    ],
    ['Версия записи трубы', String(task.pipeRevision)],
    ['Планируемый отвод', task.bendNumber],
    ['Модель станка', task.machine],
    ['Целевой угол', `${task.targetAngle}°`],
    ['Дата создания', task.date],
    ['Технологическая конфигурация', 'Не выбрана'],
    ['Режимы и допуски', 'Не заданы'],
  ];

  return (
    <Modal
      title={`Задание ${task.number}`}
      description="Характеристики исходной трубы на момент создания задания. Поздняя правка журнала их не меняет."
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
      <div className="inline-note">
        <Icon name="info" size={18} />
        <p>
          Производственные расчёты и проверка совместимости трубы со станком по
          технологической карте пока не подключены.
        </p>
      </div>
      <div className="modal-actions">
        <button className="button button--primary" onClick={onClose}>
          Закрыть
        </button>
      </div>
    </Modal>
  );
}
