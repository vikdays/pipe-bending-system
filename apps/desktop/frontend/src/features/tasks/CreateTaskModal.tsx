import { useState } from 'react';
import type { FormEvent } from 'react';
import { Modal } from '../../components/Modal';
import { Icon } from '../../components/Icon';
import type { TaskDraft } from './taskTypes';
import type { Pipe } from '../pipes/pipeTypes';
import { pipeDimensions } from '../pipes/pipeFields';

export function CreateTaskModal({
  nextNumber,
  pipeOptions,
  isSubmitting,
  error,
  onCreate,
  onClose,
}: {
  nextNumber: number;
  pipeOptions: Pipe[];
  isSubmitting: boolean;
  error: string;
  onCreate: (draft: TaskDraft) => Promise<void>;
  onClose: () => void;
}) {
  const [pipeId, setPipeId] = useState('');
  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    void onCreate({
      number: String(form.get('number')),
      pipeId,
      pipeRevision:
        pipeOptions.find((pipe) => pipe.id === pipeId)?.revision ?? 0,
      bendNumber: String(form.get('bend')),
      machine: String(form.get('machine')),
      targetAngle: String(form.get('angle')),
    });
  }

  return (
    <Modal
      title="Новое задание"
      description="Выберите конкретную трубу из трубного журнала. Её характеристики сохранятся в задании."
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
              value={pipeId}
              onChange={(event) => setPipeId(event.target.value)}
            >
              <option value="" disabled>
                Выберите трубу
              </option>
              {pipeOptions.map((pipe) => (
                <option key={pipe.id} value={pipe.id}>
                  {pipe.inventoryNumber} · зав. № {pipe.factoryNumber} ·{' '}
                  {pipeDimensions(pipe)}
                </option>
              ))}
            </select>
          </label>
          <label className="form-field">
            Модель станка
            <input
              name="machine"
              required
              maxLength={160}
              placeholder="Укажите модель по выбранной техкарте"
            />
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
            Перед созданием задания у трубы должны быть заполнены диаметр,
            толщина стенки и длина. Проверка совместимости со станком по
            технологической карте пока не подключена.
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
