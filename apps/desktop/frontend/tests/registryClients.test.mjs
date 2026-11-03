import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createPipeService } from '../src/features/pipes/pipeService.ts';
import { emptyPipeInput } from '../src/features/pipes/pipeFields.ts';
import { createTaskService } from '../src/features/tasks/taskService.ts';

const authorize = (operation) => operation('access-from-auth-client');
const pipe = (id) => ({
  ...emptyPipeInput(),
  id,
  revision: 1,
  inventoryNumber: id,
  factoryNumber: '0001',
  diameterMm: 530,
  wallThicknessMm: 8,
  lengthMm: 12000,
  certificates: [],
  createdAt: '2026-11-02T00:00:00Z',
  updatedAt: '2026-11-02T00:00:00Z',
});

test('pipe options come from the register even when there are no tasks', async () => {
  const records = [pipe('p-one'), pipe('p-two')];
  const service = createPipeService(
    {
      ListPipes: async (access) => {
        assert.equal(access, 'access-from-auth-client');
        return records;
      },
    },
    authorize,
  );
  const tasks = createTaskService({ ListTasks: async () => [] }, authorize);
  assert.deepEqual(await tasks.getTasks(), []);
  assert.deepEqual(
    (await service.getPipes()).map((item) => item.id),
    ['p-one', 'p-two'],
  );
});

test('task creation sends the selected stable ID and revision, even for duplicate factory numbers', async () => {
  const selected = pipe('p-two');
  const draft = {
    number: 'TEST-TASK',
    pipeId: selected.id,
    pipeRevision: selected.revision,
    bendNumber: 'TEST-BEND',
    machine: 'Условный станок',
    targetAngle: '9,5',
  };
  const service = createTaskService(
    {
      CreateTask: async (access, input) => {
        assert.equal(access, 'access-from-auth-client');
        assert.deepEqual(input, draft);
        return {
          ...input,
          id: 't-one',
          pipeSnapshot: selected,
          targetAngle: '9.5',
          status: 'draft',
          createdAt: '2026-11-02T00:00:00Z',
        };
      },
    },
    authorize,
  );
  const task = await service.createTask(draft);
  assert.equal(task.pipeId, 'p-two');
  assert.equal(task.pipeNumber, '0001');
  assert.equal(task.pipeInventoryNumber, 'p-two');
  assert.equal(task.pipeSnapshot.id, 'p-two');
  assert.equal(task.targetAngle, '9.5');
});

test('pipe updates carry the loaded revision and do not accidentally create a new record', async () => {
  const input = emptyPipeInput();
  const service = createPipeService(
    {
      UpdatePipe: async (access, id, revision, value) => {
        assert.equal(access, 'access-from-auth-client');
        assert.equal(id, 'p-one');
        assert.equal(revision, 3);
        assert.equal(value, input);
        return { ...pipe(id), revision: 4 };
      },
      CreatePipe: async () => {
        assert.fail('update became create');
      },
    },
    authorize,
  );
  assert.equal(
    (await service.savePipe(input, { id: 'p-one', revision: 3 })).revision,
    4,
  );
});
