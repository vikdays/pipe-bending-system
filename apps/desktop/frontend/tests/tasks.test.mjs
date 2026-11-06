import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createTaskService } from '../src/features/tasks/taskService.ts';
import { emptyTaskDraft } from '../src/features/tasks/taskTypes.ts';
import { emptyPipeInput } from '../src/features/pipes/pipeFields.ts';

const authorize = (operation) => operation('private-access');
const dto = (id, queuePosition = 1) => ({
  ...emptyTaskDraft(),
  id,
  number: id,
  pipeSnapshot: { ...emptyPipeInput(), id: 'p', revision: 2, certificates: [] },
  configurationSnapshot: null,
  revision: 3,
  queuePosition,
  status: 'draft',
  createdAt: '2026-11-03T00:00:00Z',
});
test('batch service carries the stable request ID and preserves the server execution order', async () => {
  const drafts = [
    { ...emptyTaskDraft(), pipeId: 'second' },
    { ...emptyTaskDraft(), pipeId: 'first' },
  ];
  const service = createTaskService(
    {
      CreateTasks: async (access, batch) => {
        assert.equal(access, 'private-access');
        assert.deepEqual(batch, { requestId: 'stable-request', tasks: drafts });
        return [dto('task-a', 11), dto('task-b', 12)];
      },
    },
    authorize,
  );
  const tasks = await service.createTasks('stable-request', drafts);
  assert.deepEqual(
    tasks.map((t) => [t.id, t.queuePosition]),
    [
      ['task-a', 11],
      ['task-b', 12],
    ],
  );
});
test('task updates and preparation carry the loaded revision through the native bridge', async () => {
  const calls = [];
  const transport = {
    UpdateTask: async (...args) => {
      calls.push(args);
      return dto('task');
    },
    PrepareTask: async (...args) => {
      calls.push(args);
      return {
        ...dto('task'),
        status: 'ready',
        revision: 4,
        snapshotVersion: 1,
        preparedAt: '2026-11-03T01:00:00Z',
      };
    },
  };
  const service = createTaskService(transport, authorize);
  const draft = emptyTaskDraft();
  await service.updateTask('task', 3, draft);
  const task = await service.prepareTask('task', 3);
  assert.deepEqual(calls, [
    ['private-access', 'task', 3, draft],
    ['private-access', 'task', 3],
  ]);
  assert.equal(task.status, 'ready');
  assert.equal(task.revision, 4);
});
test('missing batch response is not treated as saved drafts', async () => {
  const service = createTaskService(
    { CreateTasks: async () => null },
    authorize,
  );
  await assert.rejects(
    service.createTasks('request', [emptyTaskDraft()]),
    (error) => error.code === 'INTERNAL_ERROR',
  );
});

test('export preserves the requested revision and distinguishes cancellation, success and unconfirmed journal', async () => {
  const calls = [];
  for (const outcome of ['cancelled', 'saved', 'written_unlogged']) {
    const service = createTaskService(
      {
        ExportTasks: async (...args) => {
          calls.push(args);
          return {
            outcome,
            fileName: 'task.json',
            taskCount: 1,
            exportedTasks:
              outcome === 'saved' ? [{ id: 'task', revision: 4 }] : null,
            message: 'message',
          };
        },
      },
      authorize,
    );
    assert.equal(
      (await service.exportTasks([{ id: 'task', revision: 4 }])).outcome,
      outcome,
    );
  }
  assert.deepEqual(
    calls,
    Array(3).fill(['private-access', [{ id: 'task', revision: 4 }]]),
  );
  for (const result of [null, { outcome: 'sent' }]) {
    const service = createTaskService(
      { ExportTasks: async () => result },
      authorize,
    );
    await assert.rejects(
      service.exportTasks([{ id: 'task', revision: 4 }]),
      (e) => e.code === 'INTERNAL_ERROR',
    );
  }
});
test('exchange journal is authorized, nullable lists are empty and write errors propagate', async () => {
  const service = createTaskService(
    {
      ListExchangeEvents: async (access) => {
        assert.equal(access, 'private-access');
        return null;
      },
      ExportTasks: async () => {
        throw new Error('write failed');
      },
    },
    authorize,
  );
  assert.deepEqual(await service.getExchangeEvents(), []);
  await assert.rejects(
    service.exportTasks([{ id: 'task', revision: 4 }]),
    /write failed/,
  );
});

test('package export forwards all ten task identities without changing their revisions', async () => {
  const tasks = Array.from({ length: 10 }, (_, i) => ({
    id: `task-${i + 1}`,
    revision: 2,
  }));
  const service = createTaskService(
    {
      ExportTasks: async (access, requested) => {
        assert.equal(access, 'private-access');
        assert.deepEqual(requested, tasks);
        return {
          outcome: 'saved',
          fileName: 'tasks.json',
          taskCount: 10,
          exportedTasks: tasks,
          message: '',
        };
      },
    },
    authorize,
  );
  assert.equal((await service.exportTasks(tasks)).taskCount, 10);
  const incomplete = createTaskService(
    {
      ExportTasks: async () => ({
        outcome: 'saved',
        fileName: 'tasks.json',
        taskCount: 9,
      }),
    },
    authorize,
  );
  await assert.rejects(
    incomplete.exportTasks(tasks),
    (e) => e.code === 'INTERNAL_ERROR',
  );
});

test('successful export immediately changes badges and ready count, preserving other tasks and revisions', async () => {
  const { applyExportedTaskStates } =
    await import('../src/features/tasks/taskExportState.ts');
  const current = Array.from({ length: 10 }, (_, i) => ({
    ...dto(`task-${i + 1}`, i + 1),
    status: 'ready',
  }));
  const confirmed = current.map((t) => ({ id: t.id, revision: t.revision }));
  const updated = applyExportedTaskStates(current, confirmed);
  assert.equal(updated.filter((t) => t.status === 'ready').length, 0);
  assert.equal(updated.filter((t) => t.status === 'exported').length, 10);
  assert.deepEqual(
    updated.map((t) => [t.id, t.revision, t.pipeSnapshot]),
    current.map((t) => [t.id, t.revision, t.pipeSnapshot]),
  );
  assert.equal(current[0].status, 'ready');
  const next = { ...dto('new-task', 11), status: 'ready' };
  const waiting = applyExportedTaskStates([...current, next], confirmed);
  assert.deepEqual(
    waiting.filter((t) => t.status === 'ready').map((t) => t.id),
    ['new-task'],
  );
  assert.equal(
    applyExportedTaskStates(
      [next],
      [{ id: next.id, revision: next.revision - 1 }],
    )[0],
    next,
  );
  assert.equal(
    applyExportedTaskStates(
      [{ ...next, status: 'completed' }],
      [{ id: next.id, revision: next.revision }],
    )[0].status,
    'completed',
  );
  assert.deepEqual(applyExportedTaskStates(current, []), current);
});

test('export status loaded from Go is accepted and partial status confirmations are rejected', async () => {
  const service = createTaskService(
    {
      ListTasks: async () => [
        {
          ...dto('task'),
          status: 'exported',
          exportedAt: '2026-11-05T00:00:00Z',
        },
      ],
    },
    authorize,
  );
  const [task] = await service.getTasks();
  assert.equal(task.status, 'exported');
  assert.equal(task.exportedAt, '2026-11-05T00:00:00Z');
  for (const exportedTasks of [
    null,
    [],
    [{ id: 'different-task', revision: 4 }],
    [{ id: 'task', revision: 3 }],
  ]) {
    const broken = createTaskService(
      {
        ExportTasks: async () => ({
          outcome: 'saved',
          fileName: 'tasks.json',
          taskCount: 1,
          message: '',
          exportedTasks,
        }),
      },
      authorize,
    );
    await assert.rejects(
      broken.exportTasks([{ id: 'task', revision: 4 }]),
      (e) => e.code === 'INTERNAL_ERROR',
    );
  }
});
