import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
import contract from '../../../../../packages/contracts/tasks-v1.schema.json';
import type { TaskPackage } from './taskTypes.ts';

export const MAX_PACKAGE_BYTES = 8 * 1024 * 1024;
const ajv = new Ajv2020({ allErrors: false, strict: true });
addFormats(ajv);
const validate = ajv.compile<TaskPackage>(contract);

// Stable serialization ignores object-key order and whitespace, preserving array order.
export function canonicalJson(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(canonicalJson).join(',')}]`;
  if (value !== null && typeof value === 'object') {
    const object = value as Record<string, unknown>;
    return `{${Object.keys(object).sort().map((key) => `${JSON.stringify(key)}:${canonicalJson(object[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

export function parseTaskPackage(text: string): TaskPackage {
  if (utf8Length(text) > MAX_PACKAGE_BYTES) {
    throw new Error('Файл слишком большой. Максимальный размер пакета — 8 МБ.');
  }
  let data: unknown;
  try { data = JSON.parse(text.replace(/^\uFEFF/, '')); }
  catch { throw new Error('Не удалось прочитать JSON. Выберите файл заданий, выгруженный из приложения руководителя.'); }
  if (!validate(data)) {
    const issue = validate.errors?.[0];
    throw new Error(`Пакет не соответствует формату заданий v1. Проверьте файл на ПК.${issue ? ` Поле: ${issue.instancePath || '/'}.` : ''}`);
  }
  const ids = new Set<string>();
  let position = 0;
  for (const task of data.tasks) {
    if (![task.revision, task.queuePosition, task.pipeSnapshot.revision, task.configurationSnapshot.revision].every(Number.isSafeInteger) || ids.has(task.id.toLowerCase())) {
      throw new Error('Пакет содержит повтор UUID или некорректный номер ревизии/очереди.');
    }
    if (task.queuePosition <= position) throw new Error('Задания должны идти по возрастанию позиций очереди без повторов.');
    if (task.pipeSnapshot.wallThicknessMm * 2 >= task.pipeSnapshot.diameterMm) {
      throw new Error(`Задание ${task.number}: некорректные размеры трубы.`);
    }
    let attributes: unknown;
    try { attributes = JSON.parse(task.pipeSnapshot.attributesJson); }
    catch { throw new Error(`Задание ${task.number}: дополнительные сведения трубы повреждены.`); }
    if (!attributes || typeof attributes !== 'object' || Array.isArray(attributes)) {
      throw new Error(`Задание ${task.number}: дополнительные сведения трубы должны быть JSON-объектом.`);
    }
    if (task.configurationSnapshot.diameterMm !== task.pipeSnapshot.diameterMm) {
      throw new Error(`Задание ${task.number}: диаметры трубы и конфигурации не совпадают.`);
    }
    ids.add(task.id.toLowerCase()); position = task.queuePosition;
  }
  return data;
}

function utf8Length(text: string): number {
  let size = 0;
  for (const char of text) {
    const code = char.codePointAt(0)!;
    size += code <= 0x7f ? 1 : code <= 0x7ff ? 2 : code <= 0xffff ? 3 : 4;
  }
  return size;
}
