import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createDemoTaskService } from '../src/features/tasks/taskService.ts';

async function draftFor(service, overrides = {}) {
  const [pipe] = await service.getPipeOptions();
  return {
    number: 'ЗД-ТЕСТ',
    pipeNumber: pipe.number,
    bendNumber: 'ОТ-ТЕСТ',
    machine: 'ГТ532',
    targetAngle: '9,5',
    ...overrides,
  };
}

test('creates a draft with the selected pipe and returns it in the list', async () => {
  const service = createDemoTaskService();
  const before = await service.getTasks();
  const draft = await draftFor(service, {
    number: ' ЗД-ТЕСТ ',
    bendNumber: ' ОТ-ТЕСТ ',
  });
  const created = await service.createTask(draft);
  const tasks = await service.getTasks();

  assert.equal(tasks.length, before.length + 1);
  assert.deepEqual(tasks[0], created);
  assert.equal(created.pipeNumber, draft.pipeNumber);
  assert.equal(created.number, 'ЗД-ТЕСТ');
  assert.equal(created.bendNumber, 'ОТ-ТЕСТ');
  assert.equal(created.targetAngle, '9.5');
  assert.equal(created.status, 'draft');
  assert.ok(!before.some((task) => task.id === created.id));
});

test('returned records cannot mutate the service data', async () => {
  const service = createDemoTaskService();
  const original = await service.getTasks();
  const originalNumber = original[0].number;
  original[0].number = 'Изменено вне сервиса';
  original.splice(1);
  const again = await service.getTasks();
  assert.equal(again[0].number, originalNumber);
  assert.ok(again.length > 1);

  const created = await service.createTask(await draftFor(service));
  const id = created.id;
  created.number = 'Изменено вне сервиса';
  assert.equal(
    (await service.getTasks()).find((task) => task.id === id).number,
    'ЗД-ТЕСТ',
  );
});

test('an unknown pipe is rejected without partially adding a task', async () => {
  const service = createDemoTaskService();
  const before = await service.getTasks();
  const draft = await draftFor(service, { pipeNumber: 'НЕСУЩЕСТВУЮЩАЯ' });
  await assert.rejects(service.createTask(draft), /Выберите трубу/);
  assert.deepEqual(await service.getTasks(), before);
});

test('empty fields and a malformed angle do not change the list', async () => {
  const service = createDemoTaskService();
  const before = await service.getTasks();
  for (const fields of [
    { number: ' ' },
    { bendNumber: '' },
    { machine: ' ' },
    { targetAngle: 'не число' },
  ]) {
    await assert.rejects(service.createTask(await draftFor(service, fields)));
    assert.deepEqual(await service.getTasks(), before);
  }
});

test('each dashboard session gets an independent demo list', async () => {
  const first = createDemoTaskService();
  const second = createDemoTaskService();
  const initial = await second.getTasks();
  await first.createTask(await draftFor(first));
  assert.equal((await first.getTasks()).length, initial.length + 1);
  assert.deepEqual(await second.getTasks(), initial);
  assert.deepEqual(await createDemoTaskService().getTasks(), initial);
});
