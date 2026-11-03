import type { PipeInput } from './pipeTypes.ts';

type TextKey =
  | 'inventoryNumber'
  | 'factoryNumber'
  | 'manufacturer'
  | 'strengthClass'
  | 'steelGrade'
  | 'heatNumber'
  | 'pipeSpecification'
  | 'coatingType'
  | 'coatingSpecification'
  | 'incomingInspectionReference'
  | 'allowedCategory54'
  | 'allowedCategory74';
type NumericKey =
  | 'diameterMm'
  | 'wallThicknessMm'
  | 'lengthMm'
  | 'massKg'
  | 'carbonEquivalentPct'
  | 'testPressureMpa'
  | 'minOperatingTempC';
export type PipeField =
  | {
      key: TextKey;
      label: string;
      kind: 'text';
      required?: boolean;
      maxLength?: number;
    }
  | { key: NumericKey; label: string; kind: 'number' };
export const pipeFieldGroups: { title: string; fields: PipeField[] }[] = [
  {
    title: 'Идентификация и размеры',
    fields: [
      {
        key: 'inventoryNumber',
        label: 'Инвентарный номер записи',
        kind: 'text',
        required: true,
        maxLength: 80,
      },
      {
        key: 'factoryNumber',
        label: 'Заводской номер трубы',
        kind: 'text',
        required: true,
        maxLength: 160,
      },
      { key: 'manufacturer', label: 'Завод-изготовитель', kind: 'text' },
      { key: 'diameterMm', label: 'Наружный диаметр, мм', kind: 'number' },
      { key: 'wallThicknessMm', label: 'Толщина стенки, мм', kind: 'number' },
      { key: 'lengthMm', label: 'Длина трубы, мм', kind: 'number' },
      { key: 'massKg', label: 'Масса трубы, кг', kind: 'number' },
    ],
  },
  {
    title: 'Материал и покрытие',
    fields: [
      { key: 'strengthClass', label: 'Класс прочности', kind: 'text' },
      { key: 'steelGrade', label: 'Марка стали', kind: 'text' },
      { key: 'heatNumber', label: 'Номер плавки', kind: 'text' },
      {
        key: 'carbonEquivalentPct',
        label: 'Эквивалент углерода, %',
        kind: 'number',
      },
      {
        key: 'pipeSpecification',
        label: 'ТУ / стандарт на трубу',
        kind: 'text',
      },
      { key: 'coatingType', label: 'Тип изоляции / покрытия', kind: 'text' },
      {
        key: 'coatingSpecification',
        label: 'ТУ на изоляцию / покрытие',
        kind: 'text',
      },
    ],
  },
  {
    title: 'Входной контроль и условия эксплуатации',
    fields: [
      {
        key: 'incomingInspectionReference',
        label: 'Реквизиты документа входного контроля',
        kind: 'text',
      },
      {
        key: 'testPressureMpa',
        label: 'Испытательное давление, МПа',
        kind: 'number',
      },
      {
        key: 'minOperatingTempC',
        label: 'Минимальная температура стенки при эксплуатации, °C',
        kind: 'number',
      },
      {
        key: 'allowedCategory54',
        label: 'Допустимая категория при Рраб = 5,4 МПа (из документа)',
        kind: 'text',
      },
      {
        key: 'allowedCategory74',
        label: 'Допустимая категория при Рраб = 7,4 МПа (из документа)',
        kind: 'text',
      },
    ],
  },
];

export function parseOptionalNumber(
  value: string,
  label: string,
): number | null {
  const text = value.trim().replace(',', '.');
  if (!text) return null;
  if (!/^-?\d+(\.\d+)?$/.test(text) || !Number.isFinite(Number(text)))
    throw new Error(`«${label}»: введите число или оставьте поле пустым.`);
  return Number(text);
}

export function emptyPipeInput(): PipeInput {
  return {
    inventoryNumber: '',
    factoryNumber: '',
    manufacturer: '',
    diameterMm: null,
    wallThicknessMm: null,
    lengthMm: null,
    massKg: null,
    strengthClass: '',
    steelGrade: '',
    heatNumber: '',
    carbonEquivalentPct: null,
    pipeSpecification: '',
    coatingType: '',
    coatingSpecification: '',
    testPressureMpa: null,
    minOperatingTempC: null,
    incomingInspectionReference: '',
    allowedCategory54: '',
    allowedCategory74: '',
    notes: '',
    attributesJson: '{}',
    certificateIds: [],
    newCertificate: null,
  };
}

export function pipeDimensions(pipe: {
  diameterMm: number | null;
  wallThicknessMm: number | null;
}): string {
  const format = (value: number | null) =>
    value === null
      ? '—'
      : new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 10 }).format(
          value,
        );
  return `${format(pipe.diameterMm)} × ${format(pipe.wallThicknessMm)} мм`;
}

export interface AdditionalField {
  name: string;
  value: string;
  unit: string;
}
function isAdditionalField(item: unknown): item is AdditionalField {
  return (
    typeof item === 'object' &&
    item !== null &&
    'name' in item &&
    typeof item.name === 'string' &&
    'value' in item &&
    typeof item.value === 'string' &&
    'unit' in item &&
    typeof item.unit === 'string'
  );
}
export function readAdditionalFields(json: string): AdditionalField[] {
  const data: Record<string, unknown> = JSON.parse(json);
  if (!Array.isArray(data.customFields)) return [];
  return data.customFields.filter(isAdditionalField);
}
export function writeAdditionalFields(
  original: string,
  fields: AdditionalField[],
): string {
  const kept = fields.filter(
    (field) => field.name.trim() || field.value.trim() || field.unit.trim(),
  );
  if (kept.some((field) => !field.name.trim()))
    throw new Error(
      'У каждой дополнительной характеристики должно быть название.',
    );
  const data: Record<string, unknown> = JSON.parse(original);
  if (data.customFields !== undefined && !Array.isArray(data.customFields)) {
    if (kept.length)
      throw new Error(
        'В записи уже есть дополнительные сведения другого формата. Их нужно согласовать перед добавлением характеристик.',
      );
    return JSON.stringify(data);
  }
  const unrecognized = Array.isArray(data.customFields)
    ? data.customFields.filter((item) => !isAdditionalField(item))
    : [];
  return JSON.stringify({
    ...data,
    schemaVersion: data.schemaVersion ?? 1,
    customFields: [
      ...unrecognized,
      ...kept.map((field) => ({
        ...field,
        name: field.name.trim(),
        value: field.value,
        unit: field.unit.trim(),
      })),
    ],
  });
}
