import { useState } from 'react';
import type { FormEvent } from 'react';
import { Modal } from '../../components/Modal';
import { AdditionalFieldsEditor } from './AdditionalFieldsEditor';
import { PipeSourceDetails } from './PipeSourceDetails';
import {
  emptyPipeInput,
  parseOptionalNumber,
  pipeFieldGroups,
  readAdditionalFields,
  writeAdditionalFields,
} from './pipeFields';
import type { Certificate, Pipe, PipeInput } from './pipeTypes';

export function PipeFormModal({
  pipe,
  certificates,
  busy,
  error,
  onSave,
  onClose,
  draft,
}: {
  pipe?: Pipe;
  certificates: Certificate[];
  busy: boolean;
  error: string;
  onSave: (input: PipeInput) => Promise<void>;
  onClose: () => void;
  draft?: PipeInput;
}) {
  const initial = draft ?? pipe ?? emptyPipeInput();
  const [additional, setAdditional] = useState(() =>
    readAdditionalFields(initial.attributesJson),
  );
  const [addingCertificate, setAddingCertificate] = useState(
    Boolean(draft?.newCertificate),
  );
  const [localError, setLocalError] = useState('');

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    setLocalError('');
    try {
      const form = new FormData(event.currentTarget);
      const input = emptyPipeInput();
      for (const group of pipeFieldGroups)
        for (const field of group.fields) {
          const value = String(form.get(field.key) ?? '');
          if (field.kind === 'number')
            input[field.key] = parseOptionalNumber(value, field.label);
          else input[field.key] = value;
        }
      input.notes = String(form.get('notes') ?? '');
      input.attributesJson = writeAdditionalFields(
        initial.attributesJson,
        additional,
      );
      input.certificateIds = form.getAll('certificate').map(String);
      if (addingCertificate) {
        const pages = parseOptionalNumber(
          String(form.get('certificatePages') ?? ''),
          'Количество листов',
        );
        if (pages !== null && (!Number.isInteger(pages) || pages <= 0))
          throw new Error(
            'Количество листов должно быть положительным целым числом.',
          );
        input.newCertificate = {
          reference: String(form.get('certificateReference') ?? ''),
          number: String(form.get('certificateNumber') ?? ''),
          issuedOn: String(form.get('certificateDate') ?? ''),
          issuer: String(form.get('certificateIssuer') ?? ''),
          pageCount: pages,
        };
      }
      await onSave(input);
    } catch (failure) {
      setLocalError(
        failure instanceof Error
          ? failure.message
          : 'Проверьте заполнение формы.',
      );
    }
  }

  return (
    <Modal
      wide
      title={
        draft
          ? 'Исправить строку предпросмотра'
          : pipe
            ? `Труба ${pipe.inventoryNumber}`
            : 'Новая труба'
      }
      description={
        draft
          ? 'Изменения сохранятся в черновике импорта. Журнал изменится после отдельного подтверждения импорта.'
          : 'Запись о конкретной исходной трубе. Неизвестные характеристики оставьте пустыми.'
      }
      closeDisabled={busy}
      onClose={onClose}
    >
      <form
        className="pipe-form"
        onSubmit={(event) => {
          void submit(event);
        }}
      >
        <fieldset disabled={busy} className="pipe-form-fields">
          {pipeFieldGroups.map((group, index) => (
            <details
              className="pipe-form-section"
              key={group.title}
              open={index === 0}
            >
              <summary>{group.title}</summary>
              {index === 0 && (
                <p className="auth-help">
                  Инвентарный номер уникален внутри журнала. Заводские номера у
                  разных труб могут совпадать. Для задания понадобятся диаметр,
                  стенка и длина.
                </p>
              )}
              <div className="form-grid">
                {group.fields.map((field) => (
                  <label className="form-field" key={field.key}>
                    {field.label}
                    {field.kind === 'text' && field.required ? ' *' : ''}
                    <input
                      name={field.key}
                      defaultValue={initial[field.key] ?? ''}
                      inputMode={field.kind === 'number' ? 'decimal' : 'text'}
                      required={field.kind === 'text' && field.required}
                      maxLength={
                        field.kind === 'text' ? (field.maxLength ?? 1000) : 80
                      }
                    />
                  </label>
                ))}
              </div>
            </details>
          ))}
          <details className="pipe-form-section">
            <summary>Сертификаты качества</summary>
            <p className="auth-help">
              Выберите существующий сертификат или добавьте его реквизиты. Один
              сертификат может относиться к нескольким трубам.
            </p>
            <div className="certificate-options">
              {certificates.map((certificate) => (
                <label key={certificate.id}>
                  <input
                    type="checkbox"
                    name="certificate"
                    value={certificate.id}
                    defaultChecked={
                      draft
                        ? Boolean(
                            draft.certificateIds?.includes(certificate.id),
                          )
                        : pipe?.certificates.some(
                            (item) => item.id === certificate.id,
                          )
                    }
                  />
                  <span>
                    {certificate.reference}
                    {certificate.issuer && ` · ${certificate.issuer}`}
                  </span>
                </label>
              ))}
              {certificates.length === 0 && (
                <p className="auth-help">Сертификатов в журнале пока нет.</p>
              )}
            </div>
            <label className="certificate-toggle">
              <input
                type="checkbox"
                checked={addingCertificate}
                onChange={(event) => setAddingCertificate(event.target.checked)}
              />
              Добавить новый сертификат
            </label>
            {addingCertificate && (
              <div className="form-grid">
                <label className="form-field form-field--wide">
                  Полные реквизиты *
                  <input
                    name="certificateReference"
                    defaultValue={draft?.newCertificate?.reference ?? ''}
                    required
                    maxLength={1000}
                    placeholder="Номер, дата, количество листов — как в источнике"
                  />
                </label>
                <label className="form-field">
                  Номер
                  <input
                    name="certificateNumber"
                    maxLength={1000}
                    defaultValue={draft?.newCertificate?.number ?? ''}
                  />
                </label>
                <label className="form-field">
                  Дата выдачи
                  <input
                    name="certificateDate"
                    type="date"
                    defaultValue={draft?.newCertificate?.issuedOn ?? ''}
                  />
                </label>
                <label className="form-field">
                  Количество листов
                  <input
                    name="certificatePages"
                    defaultValue={draft?.newCertificate?.pageCount ?? ''}
                    inputMode="numeric"
                    maxLength={8}
                  />
                </label>
                <label className="form-field">
                  Кем выдан
                  <input
                    name="certificateIssuer"
                    maxLength={1000}
                    defaultValue={draft?.newCertificate?.issuer ?? ''}
                  />
                </label>
              </div>
            )}
          </details>
          <details className="pipe-form-section">
            <summary>
              Дополнительные характеристики ({additional.length})
            </summary>
            <AdditionalFieldsEditor
              fields={additional}
              onChange={setAdditional}
            />
          </details>
          <PipeSourceDetails attributesJson={initial.attributesJson} />
          <label className="form-field">
            Примечание
            <textarea
              name="notes"
              rows={3}
              maxLength={4000}
              defaultValue={initial.notes}
            />
          </label>
        </fieldset>
        {(localError || error) && (
          <p className="form-error" role="alert">
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
          <button
            type="submit"
            className="button button--primary"
            disabled={busy}
          >
            {busy
              ? 'Сохранение…'
              : draft
                ? 'Сохранить в предпросмотре'
                : 'Сохранить трубу'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
