const assert = require('node:assert/strict');
const { test } = require('node:test');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const crypto = require('node:crypto');
const { DatabaseSync } = require('node:sqlite');
const ts = require('typescript');

// Load only the storage/domain TS modules; no React Native runtime is mocked.
const previousLoader = require.extensions['.ts'];
require.extensions['.ts'] = (module, filename) => module._compile(ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true },
}).outputText, filename);
const { createTaskRepository } = require('../src/storage/taskRepository.ts');
const { parseTaskPackage } = require('../src/features/tasks/taskPackage.ts');
require.extensions['.ts'] = previousLoader;
const fixture = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../../../examples/tasks/tasks-v1.json'), 'utf8'));
const digest = async (text) => crypto.createHash('sha256').update(text).digest('hex');
const clone = () => structuredClone(fixture);
function open(file = ':memory:') {
  const native = new DatabaseSync(file);
  const driver = {
    execAsync: async (sql) => { native.exec(sql); },
    runAsync: async (sql, ...values) => native.prepare(sql).run(...values),
    getFirstAsync: async (sql, ...values) => native.prepare(sql).get(...values) ?? null,
    getAllAsync: async (sql, ...values) => native.prepare(sql).all(...values),
  };
  return { native, repository: createTaskRepository(driver, digest) };
}
test('migration, packet import and reopening a real SQLite file preserve snapshots and queue', async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'pipe-mobile-'));
  let connection;
  try {
    const file = path.join(directory, 'mobile.sqlite3');
    connection = open(file);
    await connection.repository.initialize();
    assert.deepEqual(await connection.repository.list(), []);
    assert.equal((await connection.repository.importPackage(JSON.stringify(fixture), 'tasks.json')).added, 3);
    connection.native.close(); connection = open(file);
    await connection.repository.initialize();
    assert.deepEqual((await connection.repository.list()).map((t) => t.queuePosition), [1, 2, 3]);
    assert.deepEqual((await connection.repository.get(fixture.tasks[0].id)).task, fixture.tasks[0]);
    assert.equal(await connection.repository.packageCount(), 1);
    assert.deepEqual(connection.native.prepare('PRAGMA foreign_key_check').all(), []);
    assert.throws(() => connection.native.prepare('UPDATE mobile_tasks SET revision = 99').run(), /immutable/);
  } finally { connection?.native.close(); fs.rmSync(directory, { recursive: true, force: true }); }
});
test('a renamed, formatted or key-reordered package is idempotent, also under concurrent requests', async () => {
  const { native, repository } = open();
  try {
    await repository.initialize();
    const reordered = Object.fromEntries(Object.entries(fixture).reverse());
    const results = await Promise.all([
      repository.importPackage(JSON.stringify(fixture), 'one.json'),
      repository.importPackage(JSON.stringify(reordered, null, 2), 'two.json'),
    ]);
    assert.equal(results.reduce((n, r) => n + r.added, 0), 3);
    assert.equal(results.filter((r) => r.alreadyImported).length, 1);
    assert.equal(await repository.packageCount(), 1);
    assert.equal((await repository.list()).length, 3);
  } finally { native.close(); }
});
test('a conflict after a new task rolls back the entire package and its journal entry', async () => {
  const { native, repository } = open();
  try {
    await repository.initialize();
    const original = clone(); original.tasks = [original.tasks[2]];
    await repository.importPackage(JSON.stringify(original), 'original.json');
    const conflict = clone(); conflict.tasks[2].notes = 'Different content for the same UUID/revision';
    await assert.rejects(repository.importPackage(JSON.stringify(conflict), 'conflict.json'), /Конфликт/);
    assert.equal((await repository.list()).length, 1);
    assert.equal(await repository.packageCount(), 1);
    await assert.rejects(repository.importPackage(JSON.stringify({ ...conflict, tasks: [{ ...conflict.tasks[2], revision: 999 }] }), 'new-revision.json'), /Конфликт/);
    assert.equal((await repository.get(original.tasks[0].id)).task.notes, original.tasks[0].notes);
  } finally { native.close(); }
});
test('partly overlapping packets skip existing tasks without replacing them', async () => {
  const { native, repository } = open();
  try {
    await repository.initialize();
    await repository.importPackage(JSON.stringify({ ...fixture, tasks: [fixture.tasks[0]] }), 'first.json');
    const result = await repository.importPackage(JSON.stringify(fixture), 'all.json');
    assert.deepEqual(result, { added: 2, skipped: 1, alreadyImported: false });
    assert.equal(await repository.packageCount(), 2);
    assert.equal(native.prepare('SELECT COUNT(*) AS n FROM imported_package_tasks').get().n, 4);
  } finally { native.close(); }
});
test('a different task cannot occupy an existing queue position', async () => {
  const { native, repository } = open();
  try {
    await repository.initialize();
    await repository.importPackage(JSON.stringify(fixture), 'first.json');
    const next = clone(); next.tasks = [{ ...next.tasks[0], id: '22222222-2222-4222-8222-000000000001' }];
    await assert.rejects(repository.importPackage(JSON.stringify(next), 'next.json'), /Позиция/);
    assert.equal(await repository.packageCount(), 1);
  } finally { native.close(); }
});
test('bad versions, missing snapshots, credentials, wrong types and duplicate UUIDs are rejected', () => {
  for (const change of [
    (p) => { p.schemaVersion = 2; }, (p) => { delete p.tasks[0].pipeSnapshot; },
    (p) => { p.accessToken = 'must-not-be-imported'; }, (p) => { p.tasks[0].targetAngleDeg = '9'; },
    (p) => { p.tasks[1].id = p.tasks[0].id.toUpperCase(); },
    (p) => { p.tasks.reverse(); }, (p) => { p.tasks[0].pipeSnapshot.attributesJson = '[]'; },
    (p) => { p.tasks[0].configurationSnapshot.diameterMm = 1; },
  ]) {
    const data = clone(); change(data); assert.throws(() => parseTaskPackage(JSON.stringify(data)));
  }
  assert.throws(() => parseTaskPackage('{broken'));
  assert.throws(() => parseTaskPackage(' '.repeat(8 * 1024 * 1024 + 1)), /8 МБ/);
  assert.equal(parseTaskPackage('\uFEFF' + JSON.stringify(fixture)).tasks.length, 3);
});
test('newer database versions fail safely and preserve existing data', async () => {
  const { native, repository } = open();
  try {
    native.exec('PRAGMA user_version = 2');
    await assert.rejects(repository.initialize(), /более новой/);
    assert.equal(native.prepare('PRAGMA user_version').get().user_version, 2);
  } finally { native.close(); }
});
test('a database write failure leaves no partial packet and a retry can succeed', async () => {
  const { native, repository } = open();
  try {
    await repository.initialize();
    native.exec(`CREATE TRIGGER simulate_write_failure BEFORE INSERT ON mobile_tasks
      WHEN NEW.queue_position = 2 BEGIN SELECT RAISE(ABORT, 'simulated storage failure'); END;`);
    await assert.rejects(repository.importPackage(JSON.stringify(fixture), 'tasks.json'), /simulated storage failure/);
    assert.equal((await repository.list()).length, 0);
    assert.equal(await repository.packageCount(), 0);
    native.exec('DROP TRIGGER simulate_write_failure');
    assert.equal((await repository.importPackage(JSON.stringify(fixture), 'tasks.json')).added, 3);
    assert.throws(() => native.prepare("INSERT INTO imported_package_tasks VALUES ('missing', 'missing', 1)").run(), /FOREIGN KEY/);
  } finally { native.close(); }
});
test('quoted task fields are stored as data and cannot execute SQL', async () => {
  const { native, repository } = open();
  try {
    await repository.initialize();
    const data = clone(); data.tasks[0].number = "'); DROP TABLE mobile_tasks; --";
    await repository.importPackage(JSON.stringify(data), 'quoted.json');
    assert.equal((await repository.list())[0].number, data.tasks[0].number);
    assert.equal((await repository.list()).length, 3);
  } finally { native.close(); }
});
