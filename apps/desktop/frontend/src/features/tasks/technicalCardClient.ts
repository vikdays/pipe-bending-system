import { DesktopRegistryService } from '../../../bindings/changeme';
import { authClient } from '../auth/authClient';
import { createTechnicalCardService } from './technicalCardService';
export const technicalCardService = createTechnicalCardService(
  DesktopRegistryService,
  authClient.runAuthorized,
);
