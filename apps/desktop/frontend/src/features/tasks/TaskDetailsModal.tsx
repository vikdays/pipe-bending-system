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
    ['Исходная труба', task.pipeNumber],
    ['Размеры трубы', task.dimensions],
    ['Планируемый отвод', task.bendNumber],
    ['Модель станка', task.machine],
    ['Целевой угол', `${task.targetAngle}°`],
    ['Дата создания', task.date],
    ['Технологическая конфигурация', 'Не выбрана в макете'],
    ['Режимы и допуски', 'Не заданы'],
  ];

  return (
    <Modal
      title={`Задание ${task.number}`}
      description="Вымышленные данные для проверки интерфейса."
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
          В этом макете нет производственных расчётов. Данные не подтверждают
          совместимость трубы и станка.
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
