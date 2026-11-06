import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createTechnicalCardService } from '../src/features/tasks/technicalCardService.ts';
import { matchesConfiguration } from '../src/features/tasks/configurationMatching.ts';
import { emptyTaskDraft } from '../src/features/tasks/taskTypes.ts';

const authorize = (fn) => fn('private-access');
test('DOCX selection cancellation is a no-op; selection and original IDs cross the authorized bridge', async () => {
  const calls = [];
  const service = createTechnicalCardService(
    {
      PrepareTechnicalCardImport: async (...args) => {
        calls.push(args);
        return null;
      },
      CommitTechnicalCardImport: async (...args) => {
        calls.push(args);
        return 2;
      },
      OpenTechnicalCard: async (...args) => {
        calls.push(args);
      },
    },
    authorize,
  );
  assert.equal(await service.prepare('machine-id'), null);
  assert.equal(
    await service.commit('card-id', ['source-row-1', 'source-row-2']),
    2,
  );
  await service.openOriginal('card-id');
  assert.deepEqual(calls, [
    ['private-access', 'machine-id'],
    ['private-access', 'card-id', ['source-row-1', 'source-row-2']],
    ['private-access', 'card-id'],
  ]);
});
test('missing saved technical card is an error; nullable preview collections are normalized', async () => {
  const service = createTechnicalCardService(
    {
      GetTechnicalCard: async () => null,
      PrepareTechnicalCardImport: async () => ({
        id: 'card',
        rows: null,
        warnings: null,
      }),
    },
    authorize,
  );
  await assert.rejects(
    service.get('missing'),
    (e) => e.code === 'INTERNAL_ERROR',
  );
  const preview = await service.prepare('machine');
  assert.deepEqual(preview.rows, []);
  assert.deepEqual(preview.warnings, []);
});
test('configuration matching respects machine, diameter, type and exact angle; it never interpolates or autochooses alternatives', () => {
  const task = {
    ...emptyTaskDraft(),
    machineId: 'm1',
    bendType: '1ГО',
    targetAngle: '9,0',
  };
  const cfg = {
    machineId: 'm1',
    diameterMm: 530,
    bendType: '1ГО',
    targetAngle: '9',
  };
  const pipe = { diameterMm: 530 };
  assert.equal(matchesConfiguration(cfg, task, pipe), true);
  for (const other of [
    { machineId: 'm2' },
    { diameterMm: 426 },
    { bendType: '2ГО' },
    { targetAngle: '10' },
  ])
    assert.equal(matchesConfiguration({ ...cfg, ...other }, task, pipe), false);
  assert.equal(
    matchesConfiguration(cfg, { ...task, targetAngle: '9.1' }, pipe),
    false,
  );
  assert.equal(
    matchesConfiguration({ ...cfg, id: 'alternative-2' }, task, pipe),
    true,
  );
});

test('reference ranges stay separate from operating parameters and survive edits without losing unknown fields', async () => {
  const { splitConfigurationParameters, replaceModeParameters } =
    await import('../src/features/tasks/configurationParameters.ts');
  const original = [
    { name: 'Передвижка', value: '123*', unit: 'мм' },
    {
      name: 'Толщина стенки — справочная таблица',
      value: '7 - 19',
      unit: 'мм',
    },
    { name: 'Класс прочности трубы', value: 'Условный класс режима', unit: '' },
    {
      name: 'Класс прочности — справочная таблица',
      value: 'Условный диапазон А–Б',
      unit: '',
    },
    {
      name: 'Неизвестный параметр',
      value: 'сохранить как есть*',
      unit: 'исходная единица',
    },
  ];
  const untouched = original.map((p) => ({ ...p }));
  const groups = splitConfigurationParameters(original);
  assert.deepEqual(
    groups.mode.map((p) => p.name),
    ['Передвижка', 'Класс прочности трубы', 'Неизвестный параметр'],
  );
  assert.equal(groups.reference.length, 2);
  assert.deepEqual(replaceModeParameters(original, groups.mode), original);
  const edited = groups.mode.map((p) =>
    p.name === 'Передвижка' ? { ...p, value: '124*' } : p,
  );
  const saved = replaceModeParameters(original, edited);
  assert.deepEqual(
    splitConfigurationParameters(saved).reference,
    groups.reference,
  );
  assert.equal(saved[4].value, 'сохранить как есть*');
  assert.deepEqual(original, untouched);
  assert.deepEqual(replaceModeParameters(original, []), groups.reference);
  const extra = { name: 'Ещё один параметр', value: 'текст', unit: '' };
  assert.deepEqual(
    replaceModeParameters(original, [...edited, extra]).at(-1),
    extra,
  );
});

test('reference provenance comes from the reference table, not the mode caption, and missing metadata is not invented', async () => {
  const { referenceTableOrigin } =
    await import('../src/features/tasks/configurationParameters.ts');
  const origin = referenceTableOrigin(
    JSON.stringify({
      fileName: 'fictional.docx',
      caption: 'Условная таблица режимов',
      row: 100,
      referenceTable: {
        caption: 'Условная справочная таблица',
        row: 3,
        cells: ['530', '7 - 19'],
      },
    }),
  );
  assert.deepEqual(origin, {
    fileName: 'fictional.docx',
    caption: 'Условная справочная таблица',
    row: 3,
    diameter: '530',
  });
  for (const raw of [
    '{broken',
    'null',
    '[]',
    '{}',
    '{"caption":"Только таблица режимов"}',
  ]) {
    assert.deepEqual(referenceTableOrigin(raw), {
      fileName: '',
      caption: '',
      row: null,
      diameter: '',
    });
  }
});
