import { useEffect, useState } from 'react';
import { authErrorMessage } from '../auth/appError';
import type {
  DocumentService,
  ImportConfiguration,
  ImportPreview,
  ImportSummary,
} from './documentService';
import type { PipeInput } from './pipeTypes';

export function useExcelImport(
  service: DocumentService,
  onImported: () => void,
) {
  const [history, setHistory] = useState<ImportSummary[]>([]);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<string[]>([]);
  useEffect(() => {
    let active = true;
    service
      .history()
      .then((items) => {
        if (active) setHistory(items);
      })
      .catch((e) => {
        if (active) setError(authErrorMessage(e));
      })
      .finally(() => {
        if (active) setBusy(false);
      });
    return () => {
      active = false;
    };
  }, [service]);
  async function run(operation: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError('');
    try {
      await operation();
    } catch (e) {
      setError(authErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }
  function accept(next: ImportPreview) {
    setPreview(next);
    setSelected((previous) =>
      previous.filter((id) =>
        next.rows.some((row) => row.id === id && row.status === 'new'),
      ),
    );
  }
  return {
    history,
    preview,
    busy,
    error,
    selected,
    setSelected,
    prepare: () =>
      run(async () => {
        const next = await service.prepare();
        if (next) {
          accept(next);
          setSelected([]);
          setHistory(await service.history());
        }
      }),
    resume: (id: string) =>
      run(async () => {
        accept(await service.get(id));
        setSelected([]);
      }),
    refresh: () =>
      run(async () => {
        if (preview) accept(await service.get(preview.id));
        else setHistory(await service.history());
      }),
    configure: (c: ImportConfiguration) =>
      run(async () => {
        if (preview) {
          accept(await service.configure(preview.id, preview.revision, c));
          setSelected([]);
        }
      }),
    edit: async (rowId: string, input: PipeInput) => {
      if (!preview || busy) return false;
      let saved = false;
      await run(async () => {
        accept(await service.edit(preview.id, preview.revision, rowId, input));
        saved = true;
      });
      return saved;
    },
    commit: () =>
      run(async () => {
        if (!preview) return;
        const result = await service.commit(
          preview.id,
          preview.revision,
          selected,
        );
        accept({
          ...preview,
          status: 'committed',
          result,
          revision: preview.revision + 1,
          rows: preview.rows.map((row) =>
            selected.includes(row.id) ? { ...row, status: 'imported' } : row,
          ),
        });
        setSelected([]);
        onImported();
        accept(await service.get(preview.id));
        setHistory(await service.history());
      }),
    openOriginal: () =>
      run(async () => {
        if (preview) await service.openOriginal(preview.id);
      }),
    back: () => {
      setPreview(null);
      setSelected([]);
      setError('');
    },
  };
}
