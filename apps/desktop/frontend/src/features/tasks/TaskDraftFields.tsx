import type { TaskDraft, Configuration, Machine } from './taskTypes';
import type { Pipe } from '../pipes/pipeTypes';
import { pipeDimensions } from '../pipes/pipeFields';
import { matchesConfiguration } from './configurationMatching';

export function TaskDraftFields({
  value,
  pipes,
  machines,
  configurations,
  onChange,
  onConfiguration,
}: {
  value: TaskDraft;
  pipes: Pipe[];
  machines: Machine[];
  configurations: Configuration[];
  onChange: (value: TaskDraft) => void;
  onConfiguration: (configuration?: Configuration) => void;
}) {
  const pipe = pipes.find((p) => p.id === value.pipeId);
  const config = configurations.find((c) => c.id === value.configurationId);
  const choices = configurations.filter((c) =>
    matchesConfiguration(c, value, pipe),
  );
  function change(next: TaskDraft) {
    if (
      config &&
      !matchesConfiguration(
        config,
        next,
        pipes.find((p) => p.id === next.pipeId),
      )
    ) {
      next = { ...next, configurationId: '', configurationRevision: 0 };
    }
    onChange(next);
  }
  return (
    <>
      <div className="form-grid">
        <label className="form-field">
          Номер задания
          <input
            value={value.number}
            onChange={(e) => onChange({ ...value, number: e.target.value })}
            maxLength={160}
            placeholder="Назначится при создании"
          />
        </label>
        <label className="form-field">
          Номер будущего отвода
          <input
            value={value.bendNumber}
            maxLength={160}
            onChange={(e) => onChange({ ...value, bendNumber: e.target.value })}
          />
        </label>
        <label className="form-field form-field--wide">
          Исходная труба *
          <select
            required
            value={value.pipeId}
            onChange={(e) => {
              const p = pipes.find((p) => p.id === e.target.value);
              change({
                ...value,
                pipeId: e.target.value,
                pipeRevision: p?.revision ?? 0,
              });
            }}
          >
            <option value="">Выберите трубу из журнала</option>
            {pipes.map((p) => (
              <option key={p.id} value={p.id}>
                {p.inventoryNumber} · зав. № {p.factoryNumber} ·{' '}
                {pipeDimensions(p)}
              </option>
            ))}
          </select>
        </label>
        <label className="form-field form-field--wide">
          Станок
          <select
            value={value.machineId}
            onChange={(e) => {
              const m = machines.find((m) => m.id === e.target.value);
              change({
                ...value,
                machineId: m?.id ?? '',
                machine: m?.name ?? '',
                configurationId: '',
                configurationRevision: 0,
              });
            }}
          >
            <option value="">Выберите станок</option>
            {machines.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
                {m.serialNumber ? ` · зав. № ${m.serialNumber}` : ''}
              </option>
            ))}
          </select>
        </label>
        <label className="form-field">
          Тип отвода
          <select
            value={value.bendType}
            onChange={(e) => change({ ...value, bendType: e.target.value })}
          >
            <option value="">Не выбран</option>
            <option value="1ГО">1ГО</option>
            <option value="2ГО">2ГО</option>
          </select>
        </label>
        <label className="form-field">
          Целевой угол, °
          <input
            value={value.targetAngle}
            inputMode="decimal"
            maxLength={40}
            pattern="[0-9]+([.,][0-9]+)?"
            onChange={(e) => change({ ...value, targetAngle: e.target.value })}
          />
        </label>
        <label className="form-field form-field--wide">
          Технологическая конфигурация
          <select
            disabled={!value.machineId}
            value={value.configurationId}
            onChange={(e) => {
              const c = configurations.find((c) => c.id === e.target.value);
              onChange({
                ...value,
                configurationId: c?.id ?? '',
                configurationRevision: c?.revision ?? 0,
              });
            }}
          >
            <option value="">
              {value.machineId
                ? 'Выберите табличный вариант'
                : 'Сначала выберите станок'}
            </option>
            {config && !choices.some((c) => c.id === config.id) && (
              <option value={config.id}>
                {config.name} · не соответствует выбранным данным
              </option>
            )}
            {choices.map((c) => (
              <option value={c.id} key={c.id}>
                {c.name}
                {c.needsReview ? ' · требуется сверка с ТК' : ''}
              </option>
            ))}
          </select>
        </label>
      </div>
      {value.machineId && choices.length === 0 && (
        <p className="auth-help">
          Для выбранного станка, диаметра, типа и угла нет конфигурации.
          Импортируйте техкарту через «Техкарты и конфигурации» или внесите
          сведения вручную по документу.
        </p>
      )}
      <div className="document-toolbar">
        <button
          type="button"
          className="button button--secondary"
          onClick={() => onConfiguration()}
        >
          Новая конфигурация
        </button>
        {config && (
          <button
            type="button"
            className="button button--secondary"
            onClick={() => onConfiguration(config)}
          >
            Проверить конфигурацию
          </button>
        )}
      </div>
      {config?.needsReview && (
        <p className="auth-help">
          Откройте конфигурацию, сверьте параметры и сноски с оригиналом и
          подтвердите проверку. До этого подготовка задания заблокирована.
        </p>
      )}
      {pipe && (
        <p className="auth-help">
          Труба: {pipeDimensions(pipe)} · длина {pipe.lengthMm ?? 'не указана'}
          {pipe.lengthMm !== null ? ' мм' : ''} · версия {pipe.revision}.{' '}
          {config ? `ТК: ${config.technicalCardReference}.` : ''}
        </p>
      )}
      <label className="form-field">
        Примечание к заданию
        <textarea
          value={value.notes}
          rows={2}
          maxLength={4000}
          onChange={(e) => onChange({ ...value, notes: e.target.value })}
        />
      </label>
    </>
  );
}
