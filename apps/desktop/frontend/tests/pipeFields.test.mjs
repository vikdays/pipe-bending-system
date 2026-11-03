import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  parseOptionalNumber,
  readAdditionalFields,
  writeAdditionalFields,
} from '../src/features/pipes/pipeFields.ts';

test('optional numeric fields preserve missing, zero, negative temperature and decimal commas', () => {
  assert.equal(parseOptionalNumber('', 'Размер'), null);
  assert.equal(parseOptionalNumber('  ', 'Размер'), null);
  assert.equal(parseOptionalNumber('0', 'Температура'), 0);
  assert.equal(parseOptionalNumber('-20', 'Температура'), -20);
  assert.equal(parseOptionalNumber(' 12,3 ', 'Стенка'), 12.3);
  for (const value of ['12 мм', '0x10', 'NaN', 'Infinity', '1,2,3', '1e3', '—'])
    assert.throws(() => parseOptionalNumber(value, 'Размер'));
});

test('editing additional fields preserves unrecognized data and original field provenance', () => {
  const source = JSON.stringify({
    customFields: [
      {
        name: 'Условная характеристика',
        value: '0',
        unit: '%',
        source: { page: 2 },
      },
      { futureFormat: true },
    ],
    unknown: { nested: [null, 0, 'исходный текст'] },
  });
  const fields = readAdditionalFields(source);
  fields[0] = { ...fields[0], value: '0,1' };
  const saved = JSON.parse(writeAdditionalFields(source, fields));
  assert.deepEqual(saved.unknown, JSON.parse(source).unknown);
  assert.deepEqual(saved.customFields[0], { futureFormat: true });
  assert.deepEqual(saved.customFields[1].source, { page: 2 });
  assert.equal(saved.customFields[1].value, '0,1');
  assert.equal(saved.schemaVersion, 1);
  assert.throws(() =>
    writeAdditionalFields('{}', [{ name: '', value: '5', unit: 'мм' }]),
  );
});

test('a different custom-field format is retained instead of overwritten', () => {
  const original = '{"customFields":{"future":true},"sourceVersion":2}';
  assert.deepEqual(
    JSON.parse(writeAdditionalFields(original, [])),
    JSON.parse(original),
  );
  assert.throws(() =>
    writeAdditionalFields(original, [{ name: 'Новая', value: '5', unit: '' }]),
  );
});
