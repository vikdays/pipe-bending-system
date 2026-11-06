import { useState } from 'react';
import type { FormEvent } from 'react';
import { Modal } from '../../components/Modal';
import { ConfigurationSourceDetails } from './ConfigurationSourceDetails';
import { TechnicalCardReferenceDetails } from './TechnicalCardReferenceDetails';
import {
  replaceModeParameters,
  splitConfigurationParameters,
} from './configurationParameters';
import { AdditionalFieldsEditor } from '../pipes/AdditionalFieldsEditor';
import { parseOptionalNumber } from '../pipes/pipeFields';
import type { Configuration, ConfigurationInput, Machine } from './taskTypes';

export function ConfigurationFormModal({
  configuration,
  machines,
  busy,
  error,
  onSave,
  onClose,
  onOpenOriginal,
  requireSourceReview = false,
}: {
  configuration?: Configuration;
  machines: Machine[];
  busy: boolean;
  error: string;
  onSave: (input: ConfigurationInput) => Promise<void>;
  onClose: () => void;
  onOpenOriginal: (id: string) => Promise<void>;
  requireSourceReview?: boolean;
}) {
  const [parameters, setParameters] = useState(
    () => splitConfigurationParameters(configuration?.parameters ?? []).mode,
  );
  const [localError, setLocalError] = useState('');
  async function openOriginal() {
    if (!configuration?.technicalCardId) return;
    try {
      await onOpenOriginal(configuration.technicalCardId);
    } catch (e) {
      setLocalError(
        e instanceof Error ? e.message : 'Не удалось открыть оригинал.',
      );
    }
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    setLocalError('');
    const form = new FormData(event.currentTarget);
    try {
      if (
        requireSourceReview &&
        configuration?.technicalCardId &&
        form.get('reviewed') !== 'on'
      ) {
        throw new Error(
          'Сверьте параметры и сноски с оригиналом ТК и отметьте подтверждение сверки внизу формы.',
        );
      }
      const diameter = parseOptionalNumber(
        String(form.get('diameter') ?? ''),
        'Диаметр',
      );
      if (diameter === null || diameter <= 0)
        throw new Error('Укажите положительный диаметр трубы.');
      const machineId = configuration?.technicalCardId
        ? configuration.machineId
        : String(form.get('machineId') ?? '');
      await onSave({
        name: String(form.get('name') ?? ''),
        version: String(form.get('version') ?? ''),
        machineId,
        machine: machines.find((m) => m.id === machineId)?.name ?? '',
        technicalCardReference: String(form.get('reference') ?? ''),
        diameterMm: diameter,
        parameters: replaceModeParameters(
          configuration?.parameters ?? [],
          parameters,
        ),
        notes: String(form.get('notes') ?? ''),
        bendType: String(form.get('bendType') ?? ''),
        targetAngle: String(form.get('targetAngle') ?? ''),
        minPipeLengthMm: parseOptionalNumber(
          String(form.get('minimum') ?? ''),
          'Минимальная длина',
        ),
        confirmSourceReview: form.get('reviewed') === 'on',
      });
    } catch (e) {
      setLocalError(e instanceof Error ? e.message : 'Проверьте конфигурацию.');
    }
  }
  return (
    <Modal
      wide
      title={configuration ? 'Проверить конфигурацию' : 'Новая конфигурация'}
      description="Сверьте значения, единицы и условия применения с документом. Подготовленные задания сохраняют прежний снимок при изменении каталога."
      closeDisabled={busy}
      onClose={onClose}
    >
      <form
        onSubmit={(e) => {
          void submit(e);
        }}
      >
        <fieldset disabled={busy} className="pipe-form-fields">
          {configuration?.technicalCardId && (
            <div className="source-document-note">
              {requireSourceReview && (
                <p>
                  Откройте оригинал DOCX, сравните с ним значения, единицы,
                  диапазоны и сноски. Затем внизу формы отметьте «Параметры
                  режима, справочные сведения и сноски сверены с оригиналом ТК»
                  и нажмите «Подтвердить сверку». После этого нужно сохранить
                  черновик задания.
                </p>
              )}
              <p>Источник: {configuration.technicalCardReference}</p>
              <p className="auth-help">
                {configuration.needsReview
                  ? 'Требуется сверка по оригиналу.'
                  : 'Сверка подтверждена.'}{' '}
                Год документа не подтверждает утверждение карты. Условия со
                звёздочками и диапазоны материалов нужно проверить отдельно.
              </p>
              <button
                type="button"
                className="button button--secondary"
                onClick={() => {
                  void openOriginal();
                }}
              >
                Открыть оригинал DOCX
              </button>
              <ConfigurationSourceDetails raw={configuration.sourceJson} />
            </div>
          )}
          <div className="form-grid">
            <label className="form-field form-field--wide">
              Название *
              <input
                name="name"
                required
                maxLength={1000}
                defaultValue={configuration?.name ?? ''}
              />
            </label>
            <label className="form-field">
              Версия по документу *
              <input
                name="version"
                required
                maxLength={160}
                defaultValue={configuration?.version ?? ''}
              />
            </label>
            <label className="form-field">
              Станок *
              <select
                name="machineId"
                required
                defaultValue={configuration?.machineId ?? ''}
                disabled={!!configuration?.technicalCardId}
              >
                <option value="">Выберите станок</option>
                {machines.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field form-field--wide">
              Реквизиты технологической карты *
              <input
                name="reference"
                required
                maxLength={1000}
                defaultValue={configuration?.technicalCardReference ?? ''}
              />
            </label>
            <label className="form-field">
              Диаметр трубы, мм *
              <input
                name="diameter"
                required
                inputMode="decimal"
                defaultValue={configuration?.diameterMm ?? ''}
              />
            </label>
            <label className="form-field">
              Минимальная длина трубы, мм
              <input
                name="minimum"
                inputMode="decimal"
                defaultValue={configuration?.minPipeLengthMm ?? ''}
              />
            </label>
            <label className="form-field">
              Тип отвода
              <select
                name="bendType"
                defaultValue={configuration?.bendType ?? ''}
              >
                <option value="">По документу не ограничен</option>
                <option value="1ГО">1ГО</option>
                <option value="2ГО">2ГО</option>
              </select>
            </label>
            <label className="form-field">
              Целевой угол табличного варианта, °
              <input
                name="targetAngle"
                inputMode="decimal"
                defaultValue={configuration?.targetAngle ?? ''}
              />
            </label>
          </div>
          <h3 className="configuration-parameters-title">
            Параметры выбранного режима
          </h3>
          <AdditionalFieldsEditor
            nameMaxLength={160}
            valueMaxLength={1000}
            fields={parameters}
            onChange={setParameters}
            description="Вносите значения выбранного табличного варианта из документа: углы, количество гибов, передвижку и длины участков."
          />
          <label className="form-field">
            Сноски и примечание
            <textarea
              name="notes"
              rows={4}
              maxLength={4000}
              defaultValue={configuration?.notes ?? ''}
            />
          </label>
          {configuration && (
            <TechnicalCardReferenceDetails configuration={configuration} />
          )}
          {configuration?.technicalCardId && (
            <label className="review-checkbox">
              <input
                type="checkbox"
                name="reviewed"
                required={requireSourceReview}
                defaultChecked={!configuration.needsReview}
              />
              Параметры режима, справочные сведения и сноски сверены с
              оригиналом ТК
            </label>
          )}
          {configuration?.technicalCardId && (
            <p className="auth-help">
              Подтверждается корректность переноса данных из документа.
            </p>
          )}
        </fieldset>
        {(error || localError) && (
          <p role="alert" className="form-error">
            {localError || error}
          </p>
        )}
        <div className="modal-actions">
          <button
            type="button"
            className="button button--secondary"
            disabled={busy}
            onClick={onClose}
          >
            Отмена
          </button>
          <button className="button button--primary" disabled={busy}>
            {busy
              ? 'Сохранение…'
              : requireSourceReview && configuration?.technicalCardId
                ? 'Подтвердить сверку'
                : 'Сохранить конфигурацию'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
