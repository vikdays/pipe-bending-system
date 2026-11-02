import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createAuthClient } from '../src/features/auth/authClientCore.ts';
import { AppError } from '../src/features/auth/appError.ts';

const unauthenticated = () =>
  new AppError('UNAUTHENTICATED', 'Текст ошибки может меняться.');

const user = {
  id: 'fixture-user',
  username: 'supervisor',
  role: 'admin',
  enabled: true,
};
const initial = { accessToken: 'fixture-access', accessExpiresAt: 1000, user };

function bridge(overrides = {}) {
  return {
    async NeedsSetup() {
      return false;
    },
    async Bootstrap() {
      return user;
    },
    async Login() {
      return initial;
    },
    async Refresh() {
      throw unauthenticated();
    },
    async Logout() {},
    async CurrentUser() {
      return user;
    },
    async ListUsers() {
      return [user];
    },
    async CreateUser() {
      return user;
    },
    async SetUserEnabled() {},
    async ChangePassword() {},
    ...overrides,
  };
}

test('protected requests use the token returned by Go, not a role supplied by the UI', async () => {
  const received = [];
  const client = createAuthClient(
    bridge({
      async ListUsers(access) {
        received.push(access);
        return [user];
      },
    }),
    () => 0,
  );
  await client.login('supervisor', 'fixture-password');
  await client.listUsers();
  assert.deepEqual(received, ['fixture-access']);
});

test('concurrent requests share a single refresh and use the rotated access token', async () => {
  let count = 0;
  const received = [];
  const client = createAuthClient(
    bridge({
      async Refresh() {
        count += 1;
        return { ...initial, accessToken: 'rotated', accessExpiresAt: 2000 };
      },
      async CurrentUser(access) {
        received.push(access);
        return user;
      },
    }),
    () => 990_000,
  );
  await client.login('supervisor', 'fixture-password');
  await Promise.all(Array.from({ length: 5 }, () => client.currentUser()));
  assert.equal(count, 1);
  assert.deepEqual(received, Array(5).fill('rotated'));
});

test('failed refresh notifies the application and never calls the protected method', async () => {
  let invalidations = 0;
  let calls = 0;
  const client = createAuthClient(
    bridge({
      async CurrentUser() {
        calls += 1;
        return user;
      },
    }),
  );
  const unsubscribe = client.subscribe(() => {
    invalidations += 1;
  });
  await assert.rejects(client.currentUser(), {
    code: 'UNAUTHENTICATED',
  });
  assert.equal(calls, 0);
  assert.equal(invalidations, 1);
  unsubscribe();
});

test('a refresh response arriving after logout cannot restore the old session', async () => {
  let resolveRefresh;
  let revoked = false;
  const client = createAuthClient(
    bridge({
      Refresh() {
        if (revoked) return Promise.reject(unauthenticated());
        return new Promise((resolve) => {
          resolveRefresh = resolve;
        });
      },
      async Logout() {
        revoked = true;
      },
    }),
  );
  const pending = client.restore();
  await client.logout();
  resolveRefresh(initial);
  assert.equal(await pending, null);
  await assert.rejects(client.currentUser(), {
    code: 'UNAUTHENTICATED',
  });
});

test('a new login waits for logout to finish before creating a native session', async () => {
  let finishLogout;
  let logins = 0;
  const client = createAuthClient(
    bridge({
      Logout() {
        return new Promise((resolve) => {
          finishLogout = resolve;
        });
      },
      async Login() {
        logins += 1;
        return initial;
      },
    }),
  );
  const ending = client.logout();
  const entering = client.login('supervisor', 'fixture-password');
  await Promise.resolve();
  assert.equal(logins, 0);
  finishLogout();
  await Promise.all([ending, entering]);
  assert.equal(logins, 1);
});

test('permission denial is not treated as logout, but revocation is', async () => {
  let invalidations = 0;
  const client = createAuthClient(
    bridge({
      async ListUsers() {
        throw new AppError('FORBIDDEN', 'Текст отказа в правах.');
      },
      async CurrentUser() {
        throw unauthenticated();
      },
    }),
    () => 0,
  );
  client.subscribe(() => {
    invalidations += 1;
  });
  await client.login('operator', 'fixture-password');
  await assert.rejects(client.listUsers(), { code: 'FORBIDDEN' });
  assert.equal(invalidations, 0);
  await assert.rejects(client.currentUser(), {
    code: 'UNAUTHENTICATED',
  });
  assert.equal(invalidations, 1);
});

test('successful password change clears the local session', async () => {
  let invalidations = 0;
  let received;
  const client = createAuthClient(
    bridge({
      async ChangePassword(...args) {
        received = args;
      },
    }),
    () => 0,
  );
  client.subscribe(() => {
    invalidations += 1;
  });
  await client.login('supervisor', 'fixture-password');
  await client.changePassword('old-fixture', 'new-fixture');
  assert.deepEqual(received, ['fixture-access', 'old-fixture', 'new-fixture']);
  assert.equal(invalidations, 1);
  await assert.rejects(client.currentUser(), {
    code: 'UNAUTHENTICATED',
  });
});

test('INVALID_CREDENTIALS during password change keeps the existing session', async () => {
  let invalidations = 0;
  const client = createAuthClient(
    bridge({
      async ChangePassword() {
        throw new Error(
          'Сессия завершена — этот текст не определяет поведение.',
          {
            cause: {
              code: 'INVALID_CREDENTIALS',
              message: 'Текущий пароль неверен.',
            },
          },
        );
      },
    }),
    () => 0,
  );
  client.subscribe(() => {
    invalidations += 1;
  });
  await client.login('supervisor', 'fixture-password');
  await assert.rejects(client.changePassword('wrong', 'fixture-new-password'), {
    code: 'INVALID_CREDENTIALS',
  });
  assert.equal(invalidations, 0);
  assert.deepEqual(await client.currentUser(), user);
});

test('unstructured text resembling expiration neither logs out nor exposes internals', async () => {
  let invalidations = 0;
  const client = createAuthClient(
    bridge({
      async CurrentUser() {
        throw new Error('Сессия завершена. SQL password=secret');
      },
    }),
    () => 0,
  );
  client.subscribe(() => {
    invalidations += 1;
  });
  await client.login('supervisor', 'fixture-password');
  await assert.rejects(client.currentUser(), (error) => {
    assert.equal(error.code, 'INTERNAL_ERROR');
    assert.equal('status' in error, false);
    assert.doesNotMatch(error.message, /secret|SQL/);
    return true;
  });
  assert.equal(invalidations, 0);
  assert.deepEqual(await client.listUsers(), [user]);
});

test('failed native refresh ends the local session with its actual error code', async () => {
  const reasons = [];
  const client = createAuthClient(
    bridge({
      async Refresh() {
        throw new Error('transport', {
          cause: {
            code: 'INTERNAL_ERROR',
            message: 'База временно недоступна.',
          },
        });
      },
    }),
  );
  client.subscribe((reason) => {
    reasons.push(reason);
  });
  await assert.rejects(client.currentUser(), {
    code: 'INTERNAL_ERROR',
  });
  assert.equal(reasons.length, 1);
  assert.equal(reasons[0].code, 'INTERNAL_ERROR');
  await assert.rejects(client.restore(), {
    code: 'INTERNAL_ERROR',
  });
});
