import { normalizePipe } from './pipeTypes.ts';
import type { Certificate, Pipe, PipeInput } from './pipeTypes.ts';
import type { Pipe as PipeDTO } from '../../../bindings/changeme/internal/registry/models';
export interface PipeService {
  getPipes(): Promise<Pipe[]>;
  getCertificates(): Promise<Certificate[]>;
  savePipe(
    input: PipeInput,
    existing?: Pick<Pipe, 'id' | 'revision'>,
  ): Promise<Pipe>;
}
export type Authorize = <T>(
  operation: (access: string) => Promise<T>,
) => Promise<T>;
export interface PipeTransport {
  ListPipes(access: string): Promise<PipeDTO[] | null>;
  ListCertificates(access: string): Promise<Certificate[] | null>;
  CreatePipe(access: string, input: PipeInput): Promise<PipeDTO>;
  UpdatePipe(
    access: string,
    id: string,
    revision: number,
    input: PipeInput,
  ): Promise<PipeDTO>;
}
export function createPipeService(
  transport: PipeTransport,
  authorize: Authorize,
): PipeService {
  return {
    getPipes: async () =>
      ((await authorize((access) => transport.ListPipes(access))) ?? []).map(
        normalizePipe,
      ),
    getCertificates: async () =>
      (await authorize((access) => transport.ListCertificates(access))) ?? [],
    savePipe: async (input, existing) =>
      normalizePipe(
        await authorize((access) =>
          existing
            ? transport.UpdatePipe(
                access,
                existing.id,
                existing.revision,
                input,
              )
            : transport.CreatePipe(access, input),
        ),
      ),
  };
}
