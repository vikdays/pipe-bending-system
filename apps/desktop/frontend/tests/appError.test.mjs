import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { URL } from 'node:url';
import { test } from 'node:test';
import {
  AppError,
  authErrorMessage,
  toAppError,
} from '../src/features/auth/appError.ts';

const cases = JSON.parse(
  readFileSync(
    new URL('../../../../examples/contracts/app-errors.json', import.meta.url),
    'utf8',
  ),
);

test('all public Go error payloads decode from the Wails cause envelope', () => {
  for (const payload of cases) {
    const raw = new Error('Internal transport message', {
      cause: JSON.parse(JSON.stringify(payload)),
    });
    const error = toAppError(raw);
    assert.ok(error instanceof AppError);
    assert.equal(error.code, payload.code);
    assert.equal('status' in error, false);
    assert.equal(error.message, payload.message);
    assert.equal(authErrorMessage(raw), payload.message);
  }
});

test('malformed payloads and arbitrary transport details are not displayed', () => {
  for (const error of [
    new Error('secret token /private/path'),
    new Error('transport', {
      cause: { code: 123, message: 'secret' },
    }),
    new Error('transport', {
      cause: { code: 'UNKNOWN', message: 'secret' },
    }),
    new Error('transport', {
      cause: { code: 'NOT_FOUND', message: '' },
    }),
    { cause: [{ code: 'NOT_FOUND', message: 'secret' }] },
    null,
    'secret token',
  ]) {
    const result = toAppError(error);
    assert.equal(result.code, 'INTERNAL_ERROR');
    assert.equal('status' in result, false);
    assert.doesNotMatch(result.message, /secret|private/);
  }
});
