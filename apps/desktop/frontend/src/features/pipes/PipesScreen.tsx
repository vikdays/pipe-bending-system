import { Icon } from '../../components/Icon';
import { useState } from 'react';
import { ExcelImportModal } from './ExcelImportModal';
import { CertificateDocumentsModal } from './CertificateDocumentsModal';
import type { DocumentService } from './documentService';
import { PipeFormModal } from './PipeFormModal';
import { pipeDimensions } from './pipeFields';
import { usePipeRegistry } from './usePipeRegistry';
import type { PipeService } from './pipeService';

export function PipesScreen({
  service,
  onSaved,
  documents,
}: {
  service: PipeService;
  onSaved: () => void;
  documents: DocumentService;
}) {
  const journal = usePipeRegistry(service, onSaved);
  const [dialog, setDialog] = useState<'excel' | 'certificates' | null>(null);
  function changed() {
    journal.reload();
    onSaved();
  }
  return (
    <>
      <div className="page-heading">
        <div>
          <h1>Трубный журнал</h1>
        </div>
        <div className="page-actions">
          <button
            className="button button--secondary"
            disabled={journal.loading || Boolean(journal.loadError)}
            onClick={() => setDialog('excel')}
          >
            Импорт Excel
          </button>
          <button
            className="button button--secondary"
            disabled={journal.loading || Boolean(journal.loadError)}
            onClick={() => setDialog('certificates')}
          >
            Сертификаты и PDF
          </button>
          <button
            className="button button--primary"
            disabled={journal.loading || Boolean(journal.loadError)}
            onClick={() => journal.open()}
          >
            <Icon name="plus" size={19} />
            Добавить трубу
          </button>
        </div>
      </div>
      <section
        className="panel pipe-register"
        aria-label="Трубный журнал"
        aria-busy={journal.loading}
      >
        <div className="pipe-register-toolbar">
          <label className="table-search">
            <Icon name="search" size={18} />
            <input
              aria-label="Поиск по трубному журналу"
              placeholder="Номер, изготовитель, плавка или сертификат"
              value={journal.query}
              onChange={(event) => journal.setQuery(event.target.value)}
            />
          </label>
          <span>Всего труб: {journal.pipes.length}</span>
          <button
            type="button"
            className="button button--secondary"
            disabled={journal.loading}
            onClick={journal.reload}
          >
            Обновить
          </button>
        </div>
        {journal.notice && (
          <p className="pipe-register-message" role="status">
            {journal.notice}
          </p>
        )}
        {journal.loading ? (
          <p className="pipe-register-message" role="status">
            Загрузка трубного журнала…
          </p>
        ) : journal.loadError ? (
          <div className="pipe-register-message" role="alert">
            <p>{journal.loadError}</p>
            <button
              className="button button--secondary"
              onClick={journal.reload}
            >
              Повторить загрузку
            </button>
          </div>
        ) : (
          <>
            <div className="table-scroll">
              <table className="task-table">
                <thead>
                  <tr>
                    <th scope="col">Инвентарный №</th>
                    <th scope="col">Заводской №</th>
                    <th scope="col">Размеры / длина</th>
                    <th scope="col">Изготовитель / плавка</th>
                    <th scope="col">Сертификаты</th>
                    <th scope="col">
                      <span className="sr-only">Действия</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {journal.visible.map((pipe) => (
                    <tr key={pipe.id}>
                      <td>
                        <strong>{pipe.inventoryNumber}</strong>
                      </td>
                      <td>{pipe.factoryNumber}</td>
                      <td>
                        {pipeDimensions(pipe)}
                        <small className="pipe-cell-secondary">
                          {pipe.lengthMm === null
                            ? 'Длина не указана'
                            : `${new Intl.NumberFormat('ru-RU').format(pipe.lengthMm)} мм`}
                        </small>
                      </td>
                      <td>
                        {pipe.manufacturer || 'Не указан'}
                        <small className="pipe-cell-secondary">
                          Плавка: {pipe.heatNumber || '—'}
                        </small>
                      </td>
                      <td>
                        {pipe.certificates.length
                          ? pipe.certificates.map((certificate) => (
                              <span
                                className="pipe-cell-secondary"
                                key={certificate.id}
                              >
                                {certificate.reference}
                              </span>
                            ))
                          : 'Не указаны'}
                      </td>
                      <td>
                        <button
                          className="button button--secondary"
                          aria-label={`Открыть трубу ${pipe.inventoryNumber}`}
                          onClick={() => journal.open(pipe)}
                        >
                          Открыть
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {journal.visible.length === 0 && (
              <div className="pipe-register-message">
                <p>
                  {journal.pipes.length
                    ? 'По вашему запросу трубы не найдены.'
                    : 'Трубный журнал пока пуст. Добавьте первую трубу, затем создайте задание на её основе.'}
                </p>
              </div>
            )}
          </>
        )}
      </section>
      {journal.editor && (
        <PipeFormModal
          key={journal.editor.pipe?.id ?? 'new'}
          pipe={journal.editor.pipe}
          certificates={journal.certificates}
          busy={journal.busy}
          error={journal.saveError}
          onSave={journal.save}
          onClose={journal.close}
        />
      )}
      {dialog === 'excel' && (
        <ExcelImportModal
          service={documents}
          certificates={journal.certificates}
          onImported={changed}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog === 'certificates' && (
        <CertificateDocumentsModal
          service={documents}
          certificates={journal.certificates}
          pipes={journal.pipes}
          onChanged={changed}
          onClose={() => setDialog(null)}
        />
      )}
    </>
  );
}
