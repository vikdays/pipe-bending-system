import { sessionEndedError, toAppError } from './appError.ts';
import type { AppError } from './appError.ts';

export interface AuthUser {
  id: string;
  username: string;
  role: string;
  enabled: boolean;
}

export interface AuthSession {
  accessToken: string;
  accessExpiresAt: number;
  user: AuthUser;
}

export interface AuthTransport {
  NeedsSetup(): Promise<boolean>;
  Bootstrap(username: string, password: string): Promise<AuthUser>;
  Login(username: string, password: string): Promise<AuthSession>;
  Refresh(): Promise<AuthSession>;
  Logout(): Promise<void>;
  CurrentUser(access: string): Promise<AuthUser>;
  ListUsers(access: string): Promise<AuthUser[]>;
  CreateUser(
    access: string,
    username: string,
    password: string,
    role: string,
  ): Promise<AuthUser>;
  SetUserEnabled(
    access: string,
    userID: string,
    enabled: boolean,
  ): Promise<void>;
  ChangePassword(
    access: string,
    currentPassword: string,
    newPassword: string,
  ): Promise<void>;
}

// Tokens are private in-memory state: never localStorage, sessionStorage, URLs or React props.
export function createAuthClient(transport: AuthTransport, now = Date.now) {
  let session: AuthSession | null = null;
  let generation = 0;
  let refreshing: Promise<AuthSession> | null = null;
  let loggingOut: Promise<void> | null = null;
  const listeners = new Set<(reason?: AppError) => void>();

  function invalidate(reason?: AppError) {
    generation += 1;
    session = null;
    refreshing = null;
    for (const listener of listeners) listener(reason);
  }

  async function call<T>(operation: () => Promise<T>): Promise<T> {
    try {
      return await operation();
    } catch (error) {
      throw toAppError(error);
    }
  }

  function refresh(): Promise<AuthSession> {
    if (refreshing) return refreshing;
    const expected = generation;
    const request = call(() => transport.Refresh())
      .then((next) => {
        if (generation !== expected) throw sessionEndedError();
        session = next;
        return next;
      })
      .catch((error: unknown) => {
        if (generation === expected) invalidate(toAppError(error));
        throw error;
      })
      .finally(() => {
        if (refreshing === request) refreshing = null;
      });
    refreshing = request;
    return request;
  }

  async function withAccess<T>(operation: (access: string) => Promise<T>) {
    const current =
      session && session.accessExpiresAt * 1000 > now() + 30_000
        ? session
        : await refresh();
    const expected = generation;
    try {
      return await call(() => operation(current.accessToken));
    } catch (error) {
      // Backend also checks revocation and role on every protected call.
      const failure = toAppError(error);
      if (generation === expected && failure.code === 'UNAUTHENTICATED')
        invalidate(failure);
      throw error;
    }
  }

  return {
    runAuthorized: withAccess,
    needsSetup: () => call(() => transport.NeedsSetup()),
    bootstrap: (username: string, password: string) =>
      call(() => transport.Bootstrap(username, password)),
    async login(username: string, password: string) {
      if (loggingOut) await loggingOut;
      const expected = ++generation;
      session = null;
      refreshing = null;
      const next = await call(() => transport.Login(username, password));
      if (generation !== expected) throw sessionEndedError();
      session = next;
      return next.user;
    },
    async restore() {
      try {
        return (await refresh()).user;
      } catch (error) {
        if (toAppError(error).code === 'UNAUTHENTICATED') return null;
        throw error;
      }
    },
    async logout() {
      const request = call(() => transport.Logout());
      loggingOut = request;
      invalidate();
      try {
        await request;
      } finally {
        if (loggingOut === request) loggingOut = null;
      }
    },
    currentUser: () => withAccess((access) => transport.CurrentUser(access)),
    listUsers: () => withAccess((access) => transport.ListUsers(access)),
    createUser: (username: string, password: string, role: string) =>
      withAccess((access) =>
        transport.CreateUser(access, username, password, role),
      ),
    setUserEnabled: (id: string, enabled: boolean) =>
      withAccess((access) => transport.SetUserEnabled(access, id, enabled)),
    async changePassword(currentPassword: string, newPassword: string) {
      await withAccess((access) =>
        transport.ChangePassword(access, currentPassword, newPassword),
      );
      invalidate();
    },
    subscribe(listener: (reason?: AppError) => void) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}
