import { useEffect, useState } from 'react';
import { authErrorMessage } from '../auth/appError';
import type { PipeService } from './pipeService';
import type { Certificate, Pipe, PipeInput } from './pipeTypes';

export function usePipeRegistry(service: PipeService, onSaved: () => void) {
  const [pipes, setPipes] = useState<Pipe[]>([]);
  const [certificates, setCertificates] = useState<Certificate[]>([]);
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [attempt, setAttempt] = useState(0);
  const [editor, setEditor] = useState<{ pipe?: Pipe } | null>(null);
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [notice, setNotice] = useState('');
  useEffect(() => {
    let active = true;
    Promise.all([service.getPipes(), service.getCertificates()])
      .then(([records, documents]) => {
        if (!active) return;
        setPipes(records);
        setCertificates(documents);
        setLoading(false);
      })
      .catch((error: unknown) => {
        if (active) {
          setLoadError(authErrorMessage(error));
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [service, attempt]);
  function reload() {
    setLoadError('');
    setLoading(true);
    setAttempt((value) => value + 1);
  }
  function open(pipe?: Pipe) {
    setSaveError('');
    setNotice('');
    setEditor({ pipe });
  }
  async function save(input: PipeInput) {
    if (busy) return;
    setBusy(true);
    setSaveError('');
    try {
      const pipe = await service.savePipe(input, editor?.pipe);
      setEditor(null);
      setNotice(`Труба ${pipe.inventoryNumber} сохранена.`);
      reload();
      onSaved();
    } catch (error) {
      setSaveError(authErrorMessage(error));
    } finally {
      setBusy(false);
    }
  }
  const text = query.trim().toLocaleLowerCase('ru-RU');
  const visible = pipes.filter((pipe) =>
    [
      pipe.inventoryNumber,
      pipe.factoryNumber,
      pipe.manufacturer,
      pipe.heatNumber,
      pipe.steelGrade,
      ...pipe.certificates.map((certificate) => certificate.reference),
    ].some((value) => value.toLocaleLowerCase('ru-RU').includes(text)),
  );
  return {
    pipes,
    visible,
    certificates,
    query,
    setQuery,
    loading,
    loadError,
    reload,
    editor,
    open,
    close: () => setEditor(null),
    busy,
    saveError,
    save,
    notice,
  };
}
