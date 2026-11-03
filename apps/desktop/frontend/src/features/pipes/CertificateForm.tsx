import { useState } from 'react';
import type { FormEvent } from 'react';
import type { CertificateDraft } from './pipeTypes';

export function CertificateForm({
  busy,
  onSave,
}: {
  busy: boolean;
  onSave: (input: CertificateDraft) => Promise<void>;
}) {
  const [error, setError] = useState('');
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const text = String(form.get('pages') ?? '').trim();
    const pages = text ? Number(text) : null;
    if (pages !== null && (!Number.isInteger(pages) || pages <= 0)) {
      setError('Количество листов должно быть положительным целым числом.');
      return;
    }
    setError('');
    await onSave({
      reference: String(form.get('reference') ?? ''),
      number: String(form.get('number') ?? ''),
      issuedOn: String(form.get('date') ?? ''),
      pageCount: pages,
      issuer: String(form.get('issuer') ?? ''),
    });
  }
  return (
    <form
      className="certificate-create"
      onSubmit={(event) => {
        void submit(event);
      }}
    >
      <fieldset className="pipe-form-fields" disabled={busy}>
        <h3>Новый сертификат</h3>
        <p className="auth-help">
          Введите реквизиты с оригинала. Затем прикрепите PDF и выберите
          сертификат в карточках соответствующих труб.
        </p>
        <label className="form-field">
          Полные реквизиты *<input name="reference" required maxLength={1000} />
        </label>
        <div className="form-grid">
          <label className="form-field">
            Номер
            <input name="number" maxLength={1000} />
          </label>
          <label className="form-field">
            Дата выдачи
            <input name="date" type="date" />
          </label>
          <label className="form-field">
            Количество листов
            <input name="pages" inputMode="numeric" maxLength={8} />
          </label>
          <label className="form-field">
            Кем выдан
            <input name="issuer" maxLength={1000} />
          </label>
        </div>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <button className="button button--primary">Сохранить реквизиты</button>
      </fieldset>
    </form>
  );
}
