import type {
  ImportConfiguration,
  ImportPreview as PreviewDTO,
  ImportResult,
  ImportSummary,
  PipeInput,
  CertificateDraft,
  Certificate,
} from '../../../bindings/changeme/internal/registry/models';
import type { Attachment } from '../../../bindings/changeme/internal/documents/models';
import type { Authorize } from './pipeService';
import { AppError } from '../auth/appError.ts';

export type { ImportConfiguration, ImportResult, ImportSummary, Attachment };
export type ImportPreview = Omit<
  PreviewDTO,
  'rows' | 'fields' | 'sheets' | 'columns' | 'warnings'
> & {
  rows: NonNullable<PreviewDTO['rows']>;
  fields: NonNullable<PreviewDTO['fields']>;
  sheets: NonNullable<PreviewDTO['sheets']>;
  columns: NonNullable<PreviewDTO['columns']>;
  warnings: NonNullable<PreviewDTO['warnings']>;
};
export interface DocumentService {
  prepare(): Promise<ImportPreview | null>;
  get(id: string): Promise<ImportPreview>;
  configure(
    id: string,
    revision: number,
    configuration: ImportConfiguration,
  ): Promise<ImportPreview>;
  edit(
    id: string,
    revision: number,
    rowId: string,
    input: PipeInput,
  ): Promise<ImportPreview>;
  commit(
    id: string,
    revision: number,
    selection: string[],
  ): Promise<ImportResult>;
  history(): Promise<ImportSummary[]>;
  openOriginal(id: string): Promise<void>;
  attachments(certificateId: string): Promise<Attachment[]>;
  attach(certificateId: string): Promise<Attachment | null>;
  openPDF(certificateId: string, documentId: string): Promise<void>;
  detach(certificateId: string, documentId: string): Promise<void>;
  createCertificate(input: CertificateDraft): Promise<Certificate>;
}
export interface DocumentTransport {
  PrepareExcelImport(access: string): Promise<PreviewDTO | null>;
  GetExcelImport(access: string, id: string): Promise<PreviewDTO | null>;
  ConfigureExcelImport(
    access: string,
    id: string,
    revision: number,
    configuration: ImportConfiguration,
  ): Promise<PreviewDTO | null>;
  EditExcelImportRow(
    access: string,
    id: string,
    revision: number,
    rowId: string,
    input: PipeInput,
  ): Promise<PreviewDTO | null>;
  CommitExcelImport(
    access: string,
    id: string,
    revision: number,
    selection: string[],
  ): Promise<ImportResult | null>;
  ListExcelImports(access: string): Promise<ImportSummary[] | null>;
  OpenExcelImportOriginal(access: string, id: string): Promise<void>;
  ListCertificateAttachments(
    access: string,
    id: string,
  ): Promise<Attachment[] | null>;
  AttachCertificatePDF(access: string, id: string): Promise<Attachment | null>;
  OpenCertificatePDF(
    access: string,
    id: string,
    documentId: string,
  ): Promise<void>;
  DetachCertificatePDF(
    access: string,
    id: string,
    documentId: string,
  ): Promise<void>;
  CreateCertificate(
    access: string,
    input: CertificateDraft,
  ): Promise<Certificate>;
}
function required<T>(value: T | null): T {
  if (value === null)
    throw new AppError(
      'INTERNAL_ERROR',
      'Не удалось получить результат операции. Обновите окно.',
    );
  return value;
}
function normalize(p: PreviewDTO): ImportPreview {
  return {
    ...p,
    rows: p.rows ?? [],
    fields: p.fields ?? [],
    sheets: p.sheets ?? [],
    columns: p.columns ?? [],
    warnings: p.warnings ?? [],
  };
}
export function createDocumentService(
  transport: DocumentTransport,
  authorize: Authorize,
): DocumentService {
  return {
    prepare: async () => {
      const p = await authorize((a) => transport.PrepareExcelImport(a));
      return p ? normalize(p) : null;
    },
    get: async (id) =>
      normalize(
        required(await authorize((a) => transport.GetExcelImport(a, id))),
      ),
    configure: async (id, revision, c) =>
      normalize(
        required(
          await authorize((a) =>
            transport.ConfigureExcelImport(a, id, revision, c),
          ),
        ),
      ),
    edit: async (id, revision, rowId, input) =>
      normalize(
        required(
          await authorize((a) =>
            transport.EditExcelImportRow(a, id, revision, rowId, input),
          ),
        ),
      ),
    commit: async (id, revision, selection) =>
      required(
        await authorize((a) =>
          transport.CommitExcelImport(a, id, revision, selection),
        ),
      ),
    history: async () =>
      (await authorize((a) => transport.ListExcelImports(a))) ?? [],
    openOriginal: (id) =>
      authorize((a) => transport.OpenExcelImportOriginal(a, id)),
    attachments: async (id) =>
      (await authorize((a) => transport.ListCertificateAttachments(a, id))) ??
      [],
    attach: (id) => authorize((a) => transport.AttachCertificatePDF(a, id)),
    openPDF: (id, documentId) =>
      authorize((a) => transport.OpenCertificatePDF(a, id, documentId)),
    detach: (id, documentId) =>
      authorize((a) => transport.DetachCertificatePDF(a, id, documentId)),
    createCertificate: (input) =>
      authorize((a) => transport.CreateCertificate(a, input)),
  };
}
