import type { Authorize } from '../pipes/pipeService';
import type { TechnicalCardPreview as PreviewDTO } from '../../../bindings/changeme/internal/registry/models';
import { AppError } from '../auth/appError.ts';
export type TechnicalCardPreview = Omit<PreviewDTO, 'rows' | 'warnings'> & {
  rows: NonNullable<PreviewDTO['rows']>;
  warnings: NonNullable<PreviewDTO['warnings']>;
};
export interface TechnicalCardTransport {
  PrepareTechnicalCardImport(
    access: string,
    machineId: string,
  ): Promise<PreviewDTO | null>;
  GetTechnicalCard(access: string, id: string): Promise<PreviewDTO | null>;
  ListTechnicalCards(access: string): Promise<PreviewDTO[] | null>;
  CommitTechnicalCardImport(
    access: string,
    id: string,
    keys: string[],
  ): Promise<number>;
  OpenTechnicalCard(access: string, id: string): Promise<void>;
}
const normalize = (p: PreviewDTO): TechnicalCardPreview => ({
  ...p,
  rows: p.rows ?? [],
  warnings: p.warnings ?? [],
});
export function createTechnicalCardService(
  transport: TechnicalCardTransport,
  authorize: Authorize,
) {
  return {
    prepare: async (machineId: string) => {
      const p = await authorize((a) =>
        transport.PrepareTechnicalCardImport(a, machineId),
      );
      return p ? normalize(p) : null;
    },
    get: async (id: string) => {
      const p = await authorize((a) => transport.GetTechnicalCard(a, id));
      if (!p)
        throw new AppError(
          'INTERNAL_ERROR',
          'Не удалось получить техкарту. Обновите список.',
        );
      return normalize(p);
    },
    history: async () =>
      ((await authorize((a) => transport.ListTechnicalCards(a))) ?? []).map(
        normalize,
      ),
    commit: (id: string, keys: string[]) =>
      authorize((a) => transport.CommitTechnicalCardImport(a, id, keys)),
    openOriginal: (id: string) =>
      authorize((a) => transport.OpenTechnicalCard(a, id)),
  };
}
export type TechnicalCardService = ReturnType<
  typeof createTechnicalCardService
>;
