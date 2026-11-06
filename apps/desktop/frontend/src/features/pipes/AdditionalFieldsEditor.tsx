import type { AdditionalField } from './pipeFields';

export function AdditionalFieldsEditor({
  fields,
  onChange,
  description = 'Здесь можно сохранить другие сведения из документов, включая химический состав и результаты испытаний.',
  nameMaxLength = 300,
  valueMaxLength = 2000,
}: {
  fields: AdditionalField[];
  onChange: (fields: AdditionalField[]) => void;
  description?: string;
  nameMaxLength?: number;
  valueMaxLength?: number;
}) {
  function update(index: number, key: keyof AdditionalField, value: string) {
    onChange(
      fields.map((field, i) =>
        i === index ? { ...field, [key]: value } : field,
      ),
    );
  }
  return (
    <div className="additional-fields">
      <p className="auth-help">{description}</p>
      {fields.map((field, index) => (
        <div className="additional-field-row" key={index}>
          <label className="form-field">
            Характеристика
            <input
              value={field.name}
              maxLength={nameMaxLength}
              onChange={(event) => update(index, 'name', event.target.value)}
            />
          </label>
          <label className="form-field">
            Значение
            <input
              value={field.value}
              maxLength={valueMaxLength}
              onChange={(event) => update(index, 'value', event.target.value)}
            />
          </label>
          <label className="form-field">
            Единица
            <input
              value={field.unit}
              maxLength={80}
              onChange={(event) => update(index, 'unit', event.target.value)}
            />
          </label>
          <button
            type="button"
            className="button button--secondary"
            aria-label={`Удалить характеристику ${field.name || index + 1}`}
            onClick={() => onChange(fields.filter((_, i) => i !== index))}
          >
            Удалить
          </button>
        </div>
      ))}
      <button
        type="button"
        className="button button--secondary"
        onClick={() => onChange([...fields, { name: '', value: '', unit: '' }])}
      >
        Добавить характеристику
      </button>
    </div>
  );
}
