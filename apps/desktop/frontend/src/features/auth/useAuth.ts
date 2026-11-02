import { useEffect, useState } from 'react';
import { authClient, authErrorMessage } from './authClient';
import type { AuthUser } from './authClientCore';
import { toAppError } from './appError';

type AuthState =
  | { kind: 'loading' }
  | { kind: 'unavailable' }
  | { kind: 'setup' }
  | { kind: 'signedOut'; message?: string; isError?: boolean }
  | { kind: 'signedIn'; user: AuthUser; warning?: string };

export function useAuth() {
  const [state, setState] = useState<AuthState>({ kind: 'loading' });
  const [attempt, setAttempt] = useState(0);
  const signedInUserID = state.kind === 'signedIn' ? state.user.id : null;

  useEffect(
    () =>
      authClient.subscribe((reason) =>
        setState({
          kind: 'signedOut',
          message: reason?.message,
          isError: Boolean(reason),
        }),
      ),
    [],
  );

  useEffect(() => {
    let active = true;
    async function load() {
      try {
        if (await authClient.needsSetup()) {
          if (active) setState({ kind: 'setup' });
          return;
        }
        const user = await authClient.restore();
        if (active)
          setState(user ? { kind: 'signedIn', user } : { kind: 'signedOut' });
      } catch {
        if (active) setState({ kind: 'unavailable' });
      }
    }
    void load();
    return () => {
      active = false;
    };
  }, [attempt]);

  // Periodically ask Go to revalidate the account; token refresh is single-flight.
  useEffect(() => {
    if (!signedInUserID) return;
    const timer = window.setInterval(() => {
      void authClient
        .currentUser()
        .then((user) => {
          setState((current) =>
            current.kind === 'signedIn' && current.user.id === user.id
              ? { kind: 'signedIn', user }
              : current,
          );
        })
        .catch((error: unknown) =>
          setState((current) =>
            current.kind === 'signedIn' && current.user.id === signedInUserID
              ? toAppError(error).code === 'UNAUTHENTICATED'
                ? {
                    kind: 'signedOut',
                    message: authErrorMessage(error),
                    isError: true,
                  }
                : { ...current, warning: authErrorMessage(error) }
              : current,
          ),
        );
    }, 60_000);
    return () => window.clearInterval(timer);
  }, [signedInUserID]);

  return {
    state,
    retry() {
      setState({ kind: 'loading' });
      setAttempt((current) => current + 1);
    },
    async signIn(username: string, password: string, setup: boolean) {
      if (setup) {
        await authClient.bootstrap(username, password);
      } else {
        setState({ kind: 'signedOut' });
      }
      try {
        const user = await authClient.login(username, password);
        setState({ kind: 'signedIn', user });
      } catch (error) {
        // Once bootstrap succeeded, retry normal login rather than creating a second admin.
        if (setup)
          setState({
            kind: 'signedOut',
            message: authErrorMessage(error),
            isError: true,
          });
        throw error;
      }
    },
    async signOut() {
      try {
        await authClient.logout();
      } catch (error) {
        setState({
          kind: 'signedOut',
          message: authErrorMessage(error),
          isError: true,
        });
      }
    },
  };
}
