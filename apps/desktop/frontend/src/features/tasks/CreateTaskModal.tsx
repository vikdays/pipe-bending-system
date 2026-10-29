import type { FormEvent } from 'react';
import { Modal } from '../../components/Modal';
import { Icon } from '../../components/Icon';
import type { PipeOption, TaskDraft } from './taskTypes';

export function CreateTaskModal({
  nextNumber,
  pipeOptions,
  isSubmitting,
  error,
  onCreate,
  onClose,
}: {
  nextNumber: number;
  pipeOptions: PipeOption[];
  isSubmitting: boolean;
  error: string;
  onCreate: (draft: TaskDraft) => Promise<void>;
  onClose: () => void;
}) {
  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    void onCreate({
      number: String(form.get('number')),
      pipeNumber: String(form.get('pipe')),
      bendNumber: String(form.get('bend')),
      machine: String(form.get('machine')),
      targetAngle: String(form.get('angle')),
    });
  }

  return (
    <Modal
      title="Новое задание"
      description="Демонстрационный черновик для просмотра интерфейса."
      closeDisabled={isSubmitting}
      onClose={onClose}
    >
      <form className="task-form" onSubmit={handleSubmit}>
        <div className="form-grid">
          <label className="form-field">
            Номер задания
            <input
              name="number"
              required
              maxLength={40}
              defaultValue={`ЗД-${String(nextNumber).padStart(4, '0')}`}
            />
          </label>
          <label className="form-field">
            Номер будущего отвода
            <input
              name="bend"
              required
              maxLength={40}
              placeholder="Например, ОТ-007"
            />
          </label>
          <label className="form-field form-field--wide">
            Исходная труба
            <select
              name="pipe"
              required
              defaultValue={pipeOptions[0]?.number ?? ''}
            >
              {pipeOptions.map((pipe) => (
                <option key={pipe.number} value={pipe.number}>
                  {pipe.number} · {pipe.dimensions}
                </option>
              ))}
            </select>
          </label>
          <label className="form-field">
            Модель станка
            <select name="machine" defaultValue="ГТ532">
              <option>ГТ532</option>
              <option>ГТ1022</option>
            </select>
          </label>
          <label className="form-field">
            Целевой угол, °
            <input
              name="angle"
              required
              inputMode="decimal"
              pattern="[0-9]+([.,][0-9]+)?"
              placeholder="Например, 9"
            />
          </label>
        </div>
        <div className="inline-note">
          <Icon name="info" size={18} />
          <p>
            Это макет формы. Совместимость трубы и станка не проверяется.
            Черновик хранится только до обновления страницы.
          </p>
        </div>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <div className="modal-actions">
          <button
            type="button"
            className="button button--secondary"
            disabled={isSubmitting}
            onClick={onClose}
          >
            Отмена
          </button>
          <button
            type="submit"
            className="button button--primary"
            disabled={isSubmitting || pipeOptions.length === 0}
          >
            <Icon name="plus" size={18} />
            {isSubmitting ? 'Добавление…' : 'Добавить черновик'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
