import type { ConfigurationParameter } from '../../../bindings/changeme/internal/registry/models';

// Exact names written by the two studied DOCX importers, including existing
// saved snapshots. A strength-class parameter of a mode is NOT a reference row.
const referenceLabels: Record<string, string> = {
  'Толщина стенки — справочная таблица': 'Толщина стенки по справочной таблице',
  'Класс прочности — справочная таблица':
    'Класс прочности по справочной таблице',
  'Условный предел текучести — справочная таблица':
    'Условный предел текучести по справочной таблице',
  'Минимальный радиус гибки — справочная таблица':
    'Минимальный радиус гибки по ТК',
  'Целевые углы — справочная таблица': 'Диапазон целевых углов по ТК',
};
export function referenceParameterLabel(name: string): string | undefined {
  return Object.hasOwn(referenceLabels, name)
    ? referenceLabels[name]
    : undefined;
}
export function splitConfigurationParameters(
  parameters: ConfigurationParameter[],
) {
  const mode: ConfigurationParameter[] = [];
  const reference: ConfigurationParameter[] = [];
  for (const parameter of parameters) {
    (referenceParameterLabel(parameter.name) ? reference : mode).push(
      parameter,
    );
  }
  return { mode, reference };
}

// Only the operating parameters are edited. Keep reference values and unknown
// parameters intact, preserving original positions when the mode list is unchanged.
export function replaceModeParameters(
  original: ConfigurationParameter[],
  mode: ConfigurationParameter[],
) {
  let index = 0;
  const result: ConfigurationParameter[] = [];
  for (const parameter of original) {
    if (referenceParameterLabel(parameter.name)) result.push(parameter);
    else if (index < mode.length) result.push(mode[index++]);
  }
  return [...result, ...mode.slice(index)];
}
function record(value: unknown): Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}
export function referenceTableOrigin(raw: string) {
  let source: Record<string, unknown> = {};
  try {
    source = record(JSON.parse(raw));
  } catch {
    /* Older snapshots may lack provenance. */
  }
  const table = record(source.referenceTable);
  return {
    fileName: typeof source.fileName === 'string' ? source.fileName : '',
    caption: typeof table.caption === 'string' ? table.caption : '',
    row:
      typeof table.row === 'number' &&
      Number.isInteger(table.row) &&
      table.row > 0
        ? table.row
        : null,
    diameter:
      Array.isArray(table.cells) && typeof table.cells[0] === 'string'
        ? table.cells[0]
        : '',
  };
}
