import type { Pipe as PipeDTO } from '../../../bindings/changeme/internal/registry/models';
export type {
  Certificate,
  CertificateDraft,
  PipeInput,
} from '../../../bindings/changeme/internal/registry/models';
export type Pipe = Omit<PipeDTO, 'certificates'> & {
  certificates: NonNullable<PipeDTO['certificates']>;
};
export function normalizePipe(pipe: PipeDTO): Pipe {
  return { ...pipe, certificates: pipe.certificates ?? [] };
}
