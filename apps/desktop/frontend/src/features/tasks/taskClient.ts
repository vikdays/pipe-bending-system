import { DesktopRegistryService } from '../../../bindings/changeme';
import { authClient } from '../auth/authClient';
import { createTaskService } from './taskService';
export const taskService = createTaskService(
  DesktopRegistryService,
  authClient.runAuthorized,
);
