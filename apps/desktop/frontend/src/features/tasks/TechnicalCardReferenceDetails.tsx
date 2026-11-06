import { useState } from 'react';
import { authErrorMessage } from '../auth/appError';
import {
  referenceParameterLabel,
  referenceTableOrigin,
  splitConfigurationParameters,
} from './configurationParameters';
import type { Configuration } from './taskTypes';

type ReferenceConfiguration = Pick<
  Configuration,
  'parameters' | 'sourceJson' | 'technicalCardReference' | 'technicalCardId'
>;

export function TechnicalCardReferenceDetails({
  configuration,
  onOpenOriginal,
}: {
  configuration: ReferenceConfiguration;
  onOpenOriginal?: (id: string) => Promise<void>;
}) {
  const [opening, setOpening] = useState(false);
  const [error, setError] = useState('');
  const { reference } = splitConfigurationParameters(configuration.parameters);
  if (!reference.length) return null;
  const origin = referenceTableOrigin(configuration.sourceJson);
  async function openOriginal() {
    if (!onOpenOriginal || !configuration.technicalCardId || opening) return;
    setOpening(true);
    setError('');
    try {
      await onOpenOriginal(configuration.technicalCardId);
    } catch (e) {
      setError(authErrorMessage(e));
    } finally {
      setOpening(false);
    }
  }
  return (
    <section className="technical-card-reference">
      <h3>Справочные сведения из ТК</h3>
      <p className="auth-help">
        Здесь указаны диапазоны и справочные значения для группы труб.
        Фактические характеристики конкретной трубы хранятся в трубном журнале.
        Автоматическая проверка соответствия этим диапазонам пока не
        выполняется.
      </p>
      <p className="reference-origin">
        <strong>Источник: </strong>
        {origin.caption ||
          'Положение справочной таблицы не сохранено — сверьте с оригиналом ТК.'}
        {origin.row !== null && ` · строка ${origin.row}`}
        {origin.diameter &&
          ` · наружный диаметр в строке: ${origin.diameter} мм`}
      </p>
      <p className="auth-help">
        {origin.fileName ||
          configuration.technicalCardReference ||
          'Документ не указан'}
      </p>
      <dl className="detail-grid">
        {reference.map((p, index) => (
          <div key={`${p.name}-${index}`}>
            <dt>{referenceParameterLabel(p.name)}</dt>
            <dd>
              {p.value}
              {p.unit ? ` ${p.unit}` : ''}
            </dd>
          </div>
        ))}
      </dl>
      {onOpenOriginal && configuration.technicalCardId && (
        <button
          type="button"
          className="button button--secondary"
          disabled={opening}
          onClick={() => {
            void openOriginal();
          }}
        >
          {opening ? 'Открытие…' : 'Открыть оригинал ТК'}
        </button>
      )}
      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}
