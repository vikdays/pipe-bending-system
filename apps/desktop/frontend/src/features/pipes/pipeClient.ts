import { DesktopRegistryService } from '../../../bindings/changeme';
import { authClient } from '../auth/authClient';
import { createPipeService } from './pipeService';
export const pipeService = createPipeService(
  DesktopRegistryService,
  authClient.runAuthorized,
);
