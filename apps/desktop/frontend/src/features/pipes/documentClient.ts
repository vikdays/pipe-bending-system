import { DesktopRegistryService } from '../../../bindings/changeme';
import { authClient } from '../auth/authClient';
import { createDocumentService } from './documentService';
export const documentService = createDocumentService(
  DesktopRegistryService,
  authClient.runAuthorized,
);
