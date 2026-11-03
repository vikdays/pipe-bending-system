import { useEffect, useState } from 'react';
import { Modal } from '../../components/Modal';
import { authErrorMessage } from '../auth/appError';
import { CertificateForm } from './CertificateForm';
import type { DocumentService, Attachment } from './documentService';
import type { Certificate, Pipe } from './pipeTypes';

function size(bytes: number) {
  return `${new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 }).format(bytes / 1024 / 1024)} МБ`;
}
export function CertificateDocumentsModal({
  service,
  certificates,
  pipes,
  onChanged,
  onClose,
}: {
  service: DocumentService;
  certificates: Certificate[];
  pipes: Pipe[];
  onChanged: () => void;
  onClose: () => void;
}) {
  const [items, setItems] = useState(certificates);
  const [selected, setSelected] = useState(certificates[0]?.id ?? '');
  const [query, setQuery] = useState('');
  const [creating, setCreating] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [refresh, setRefresh] = useState(0);
  const [loaded, setLoaded] = useState<{
    id: string;
    attachments: Attachment[];
    error: string;
  }>({ id: '', attachments: [], error: '' });
  const [detaching, setDetaching] = useState('');
  useEffect(() => {
    let active = true;
    if (selected)
      service
        .attachments(selected)
        .then((attachments) => {
          if (active) setLoaded({ id: selected, attachments, error: '' });
        })
        .catch((e) => {
          if (active)
            setLoaded({
              id: selected,
              attachments: [],
              error: authErrorMessage(e),
            });
        });
    return () => {
      active = false;
    };
  }, [selected, service, refresh]);
  async function run(operation: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await operation();
    } catch (e) {
      setError(authErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }
  const certificate = items.find((c) => c.id === selected);
  const linked = pipes.filter((pipe) =>
    pipe.certificates.some((c) => c.id === selected),
  );
  const waiting = Boolean(selected && loaded.id !== selected);
  return (
    <Modal
      extraWide
      title="Сертификаты и оригиналы PDF"
      description="Вы сами выбираете, к какому сертификату относится PDF. Один сертификат можно связать с несколькими трубами через их карточки."
      closeDisabled={busy}
      onClose={onClose}
    >
      <div className="document-toolbar">
        <label className="form-field">
          Поиск сертификата
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Номер, дата или издатель"
          />
        </label>
        <button
          className="button button--secondary"
          disabled={busy}
          onClick={() => {
            setCreating(!creating);
            setError('');
          }}
        >
          {creating ? 'Отменить создание' : 'Новый сертификат'}
        </button>
      </div>
      {(error || (loaded.id === selected && loaded.error)) && (
        <p className="form-error" role="alert">
          {error || loaded.error}
        </p>
      )}
      {notice && (
        <p className="import-result" role="status">
          {notice}
        </p>
      )}
      {creating ? (
        <CertificateForm
          busy={busy}
          onSave={(input) =>
            run(async () => {
              const next = await service.createCertificate(input);
              setItems((previous) => [...previous, next]);
              setSelected(next.id);
              setCreating(false);
              onChanged();
            })
          }
        />
      ) : (
        <div className="certificate-workspace">
          <div className="certificate-list" aria-label="Сертификаты">
            {items
              .filter((c) =>
                `${c.reference} ${c.issuer}`
                  .toLocaleLowerCase('ru-RU')
                  .includes(query.toLocaleLowerCase('ru-RU')),
              )
              .map((c) => (
                <button
                  key={c.id}
                  className={`certificate-list-item ${selected === c.id ? 'is-selected' : ''}`}
                  disabled={busy}
                  aria-pressed={selected === c.id}
                  onClick={() => {
                    setSelected(c.id);
                    setDetaching('');
                    setError('');
                    setNotice('');
                  }}
                >
                  <strong>{c.reference}</strong>
                  <small>{c.issuer || 'Издатель не указан'}</small>
                </button>
              ))}
            {items.length === 0 && (
              <p>
                Сертификатов пока нет. Создайте реквизиты или импортируйте их
                вместе с трубами.
              </p>
            )}
          </div>
          {certificate && (
            <section
              className="certificate-detail"
              aria-label="Оригиналы выбранного сертификата"
            >
              <h3>{certificate.reference}</h3>
              <dl className="certificate-metadata">
                <div>
                  <dt>Номер</dt>
                  <dd>{certificate.number || '—'}</dd>
                </div>
                <div>
                  <dt>Дата выдачи</dt>
                  <dd>{certificate.issuedOn || '—'}</dd>
                </div>
                <div>
                  <dt>Листов по реквизитам</dt>
                  <dd>{certificate.pageCount ?? '—'}</dd>
                </div>
                <div>
                  <dt>Кем выдан</dt>
                  <dd>{certificate.issuer || '—'}</dd>
                </div>
              </dl>
              <p className="auth-help">
                Связанные трубы: {linked.length}.{' '}
                {linked
                  .slice(0, 8)
                  .map((pipe) => pipe.inventoryNumber)
                  .join(', ')}
                {linked.length > 8 ? '…' : ''}
              </p>
              <p className="auth-help">
                PDF сохраняется целиком, без распознавания и автоматического
                изменения характеристик. Если сертификат многостраничный,
                выберите полный файл. Максимальный размер — 50 МБ.
              </p>
              <div className="document-toolbar">
                <button
                  className="button button--primary"
                  disabled={busy || waiting}
                  onClick={() => {
                    void run(async () => {
                      const attached = await service.attach(selected);
                      if (attached) {
                        setLoaded({ id: '', attachments: [], error: '' });
                        setRefresh((value) => value + 1);
                        setNotice(
                          'Оригинал PDF сохранён и связан с сертификатом.',
                        );
                      }
                    });
                  }}
                >
                  Прикрепить PDF
                </button>
                <button
                  className="button button--secondary"
                  disabled={busy || waiting}
                  onClick={() => {
                    setLoaded({ id: '', attachments: [], error: '' });
                    setRefresh((value) => value + 1);
                  }}
                >
                  Обновить файлы
                </button>
              </div>
              {waiting ? (
                <p role="status">Загрузка файлов…</p>
              ) : loaded.attachments.length === 0 ? (
                <p>Оригинал PDF ещё не прикреплён.</p>
              ) : (
                loaded.attachments.map((file) => (
                  <article key={file.id} className="document-file">
                    <strong>{file.displayName}</strong>
                    <p>
                      {size(file.sizeBytes)} · страниц в PDF:{' '}
                      {file.pageCount ?? '—'}
                    </p>
                    {certificate.pageCount !== null &&
                      file.pageCount !== certificate.pageCount && (
                        <p className="auth-help">
                          Число страниц PDF отличается от реквизитов. Проверьте
                          комплектность выбранного оригинала.
                        </p>
                      )}
                    <details>
                      <summary>Контрольная сумма SHA-256</summary>
                      <code className="document-hash">{file.sha256}</code>
                    </details>
                    <div className="document-toolbar">
                      <button
                        className="button button--secondary"
                        disabled={busy}
                        onClick={() => {
                          void run(() => service.openPDF(selected, file.id));
                        }}
                      >
                        Открыть PDF
                      </button>
                      <button
                        className="button button--secondary"
                        disabled={busy}
                        onClick={() => setDetaching(file.id)}
                      >
                        Снять привязку
                      </button>
                    </div>
                    {detaching === file.id && (
                      <div className="document-detach">
                        <p>
                          Снять привязку к этому сертификату? Оригинал останется
                          в хранилище и у других сертификатов.
                        </p>
                        <div className="document-toolbar">
                          <button
                            className="button button--secondary"
                            disabled={busy}
                            onClick={() => {
                              void run(async () => {
                                await service.detach(selected, file.id);
                                setDetaching('');
                                setLoaded({
                                  id: '',
                                  attachments: [],
                                  error: '',
                                });
                                setRefresh((value) => value + 1);
                                setNotice(
                                  'Привязка снята. Другие сертификаты не изменены.',
                                );
                              });
                            }}
                          >
                            Подтвердить снятие
                          </button>
                          <button
                            className="button button--secondary"
                            disabled={busy}
                            onClick={() => setDetaching('')}
                          >
                            Отмена
                          </button>
                        </div>
                      </div>
                    )}
                  </article>
                ))
              )}
            </section>
          )}
        </div>
      )}
      {busy && <p role="status">Обработка документа…</p>}
      <div className="modal-actions">
        <button
          className="button button--secondary"
          disabled={busy}
          onClick={onClose}
        >
          Закрыть
        </button>
      </div>
    </Modal>
  );
}
