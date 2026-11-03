function record(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}
function text(value: unknown) {
  return value === null || value === undefined
    ? '—'
    : typeof value === 'object'
      ? JSON.stringify(value)
      : String(value);
}

export function PipeSourceDetails({
  attributesJson,
}: {
  attributesJson: string;
}) {
  let source: Record<string, unknown>;
  try {
    const value: unknown = JSON.parse(attributesJson);
    if (!record(value) || !record(value.source)) return null;
    source = value.source;
  } catch {
    return null;
  }
  const cells = record(source.cells) ? source.cells : {};
  return (
    <details className="pipe-form-section">
      <summary>Источник и исходные ячейки</summary>
      <p className="auth-help">
        {text(source.fileName)} · лист {text(source.sheet)} · строка{' '}
        {text(source.row)}. Исправления характеристик не меняют сохранённый
        оригинал.
      </p>
      <div className="table-scroll source-cells">
        <table className="task-table">
          <thead>
            <tr>
              <th>Ячейка</th>
              <th>Исходное значение</th>
              <th>Отображение / сохранённый результат</th>
              <th>Формула из файла</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(cells).map(
              ([axis, value]) =>
                record(value) && (
                  <tr key={axis}>
                    <td>{axis}</td>
                    <td>{text(value.raw)}</td>
                    <td>{text(value.display ?? value.cached)}</td>
                    <td>{text(value.formula)}</td>
                  </tr>
                ),
            )}
          </tbody>
        </table>
      </div>
      <details>
        <summary>SHA-256 оригинала</summary>
        <code className="document-hash">{text(source.sha256)}</code>
      </details>
    </details>
  );
}
