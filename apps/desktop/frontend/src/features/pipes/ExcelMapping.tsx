import { useState } from 'react';
import type { ImportConfiguration, ImportPreview } from './documentService';

const unitLabels: Record<string, string> = {
  mm: 'мм',
  m: 'м',
  kg: 'кг',
  t: 'т',
  MPa: 'МПа',
  percent: '%',
  C: '°C',
};
export function ExcelMapping({
  preview,
  busy,
  onApply,
}: {
  preview: ImportPreview;
  busy: boolean;
  onApply: (configuration: ImportConfiguration) => void;
}) {
  const [configuration, setConfiguration] = useState<ImportConfiguration>(() =>
    structuredClone(preview.configuration),
  );
  return (
    <details className="pipe-form-section">
      <summary>Лист, столбцы и единицы измерения</summary>
      <p className="auth-help">
        Повторное чтение заменит исправления строк значениями из оригинала.
        Журнал при этом не изменится. После смены листа или строки заголовков
        нажмите «Перечитать»: столбцы будут предложены заново.
      </p>
      <fieldset disabled={busy} className="pipe-form-fields">
        <div className="form-grid">
          <label className="form-field">
            Лист
            <select
              value={configuration.sheet}
              onChange={(e) =>
                setConfiguration({
                  ...configuration,
                  sheet: e.target.value,
                  columns: {},
                })
              }
            >
              {preview.sheets.map((sheet) => (
                <option key={sheet}>{sheet}</option>
              ))}
            </select>
          </label>
          <label className="form-field">
            Строка заголовков
            <input
              type="number"
              min={1}
              max={50000}
              value={configuration.headerRow}
              onChange={(e) =>
                setConfiguration({
                  ...configuration,
                  headerRow: Number(e.target.value),
                  columns: {},
                })
              }
            />
          </label>
        </div>
        <div className="import-mapping">
          {preview.fields.map((field) => (
            <div className="import-mapping-row" key={field.key}>
              <label className="form-field">
                {field.label}
                {field.required ? ' *' : ''}
                <select
                  aria-label={`Столбец: ${field.label}`}
                  value={configuration.columns?.[field.key] ?? 0}
                  onChange={(e) =>
                    setConfiguration({
                      ...configuration,
                      columns: {
                        ...configuration.columns,
                        [field.key]: Number(e.target.value),
                      },
                    })
                  }
                >
                  <option value={0}>Не импортировать</option>
                  {preview.columns.map((col) => (
                    <option key={col.index} value={col.index}>
                      {col.letter} — {col.label || 'Без заголовка'}
                    </option>
                  ))}
                </select>
              </label>
              {Boolean(field.units?.length) && (
                <label className="form-field">
                  Единицы в файле
                  <select
                    value={configuration.units?.[field.key] ?? field.units?.[0]}
                    onChange={(e) =>
                      setConfiguration({
                        ...configuration,
                        units: {
                          ...configuration.units,
                          [field.key]: e.target.value,
                        },
                      })
                    }
                  >
                    {field.units?.map((unit) => (
                      <option key={unit} value={unit}>
                        {unitLabels[unit] ?? unit}
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>
          ))}
        </div>
        <button
          className="button button--secondary"
          type="button"
          onClick={() => onApply(configuration)}
        >
          Перечитать оригинал с этими настройками
        </button>
      </fieldset>
    </details>
  );
}
