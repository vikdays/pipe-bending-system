import { DesktopAuthService } from '../../../bindings/changeme';
import { createAuthClient } from './authClientCore';

export const authClient = createAuthClient({
  ...DesktopAuthService,
  ListUsers: async (access) =>
    (await DesktopAuthService.ListUsers(access)) ?? [],
});

export { authErrorMessage } from './appError';
