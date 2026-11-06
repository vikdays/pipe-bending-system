function text(value: unknown): string {
  return typeof value === 'string' ? value : '';
}
function object(value: unknown): Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}
export function ConfigurationSourceDetails({ raw }: { raw: string }) {
  let source: Record<string, unknown>;
  try {
    source = object(JSON.parse(raw));
  } catch {
    return null;
  }
  const cells = Array.isArray(source.cells) ? source.cells : [];
  const pairs: { label: string; value: string }[] = [];
  if (Array.isArray(cells[0]) && Array.isArray(cells[1])) {
    const values = cells[1];
    cells[0].forEach((label, index) =>
      pairs.push({
        label: text(label),
        value: text(values[index]) || 'Объединённая ячейка — см. оригинал',
      }),
    );
  } else {
    cells.forEach((row) => {
      const cell = object(row);
      if (text(cell.label) && text(cell.value))
        pairs.push({ label: text(cell.label), value: text(cell.value) });
    });
  }
  return (
    <details>
      <summary>Расположение в документе и исходные значения</summary>
      <p className="auth-help">
        {text(source.caption)}
        {typeof source.row === 'number' ? ` · строка ${source.row}` : ''}
        {typeof source.column === 'number' && source.column > 0
          ? ` · столбец ${source.column}`
          : ''}
      </p>
      <dl className="detail-grid">
        {pairs.map((pair, index) => (
          <div key={index}>
            <dt>{pair.label}</dt>
            <dd>{pair.value}</dd>
          </div>
        ))}
      </dl>
    </details>
  );
}
