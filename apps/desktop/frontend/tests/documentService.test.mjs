import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createDocumentService } from '../src/features/pipes/documentService.ts';

test('Excel picker cancellation does not create a preview or retry selection', async () => {
  let calls = 0;
  const service = createDocumentService(
    {
      PrepareExcelImport: async (access) => {
        assert.equal(access, 'private-access');
        calls++;
        return null;
      },
    },
    (operation) => operation('private-access'),
  );
  assert.equal(await service.prepare(), null);
  assert.equal(calls, 1);
});
test('draft revisions, row selection and document IDs cross the authorized bridge', async () => {
  const calls = [];
  const result = { importId: 'i-one', added: 2 };
  const transport = {
    CommitExcelImport: async (...args) => {
      calls.push(args);
      return result;
    },
    OpenCertificatePDF: async (...args) => {
      calls.push(args);
    },
    DetachCertificatePDF: async (...args) => {
      calls.push(args);
    },
    GetExcelImport: async () => ({
      id: 'i-one',
      rows: null,
      fields: null,
      warnings: null,
      sheets: null,
      columns: null,
    }),
  };
  const service = createDocumentService(transport, (operation) =>
    operation('private-access'),
  );
  assert.equal(await service.commit('i-one', 3, ['r1', 'r2']), result);
  await service.openPDF('c-one', 'd-one');
  await service.detach('c-one', 'd-one');
  assert.deepEqual(calls, [
    ['private-access', 'i-one', 3, ['r1', 'r2']],
    ['private-access', 'c-one', 'd-one'],
    ['private-access', 'c-one', 'd-one'],
  ]);
  assert.deepEqual((await service.get('i-one')).rows, []);
});
test('a missing commit result is an error, never reported as successful import', async () => {
  const service = createDocumentService(
    { CommitExcelImport: async () => null },
    (operation) => operation('access'),
  );
  await assert.rejects(
    service.commit('i', 1, ['row']),
    (error) => error.code === 'INTERNAL_ERROR',
  );
});
