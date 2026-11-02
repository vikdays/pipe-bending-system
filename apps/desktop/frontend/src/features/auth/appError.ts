const errorCodes = [
  'VALIDATION_ERROR',
  'INVALID_CREDENTIALS',
  'UNAUTHENTICATED',
  'FORBIDDEN',
  'NOT_FOUND',
  'CONFLICT',
  'RATE_LIMITED',
  'INTERNAL_ERROR',
] as const;

export type ErrorCode = (typeof errorCodes)[number];

export class AppError extends Error {
  readonly code: ErrorCode;

  constructor(code: ErrorCode, message: string) {
    super(message);
    this.name = 'AppError';
    this.code = code;
  }
}

function isErrorCode(value: unknown): value is ErrorCode {
  return typeof value === 'string' && errorCodes.some((code) => code === value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// Wails puts the Go marshaler's payload in RuntimeError.cause.
// Never classify errors by translated text or display arbitrary transport internals.
export function toAppError(error: unknown): AppError {
  if (error instanceof AppError) return error;
  const payload = isRecord(error) ? error.cause : undefined;
  if (isRecord(payload) && isErrorCode(payload.code)) {
    if (
      typeof payload.message === 'string' &&
      payload.message.length > 0 &&
      payload.message.length <= 2000
    ) {
      return new AppError(payload.code, payload.message);
    }
  }
  return new AppError(
    'INTERNAL_ERROR',
    'Внутренняя ошибка приложения. Повторите попытку позже.',
  );
}

export function authErrorMessage(error: unknown): string {
  return toAppError(error).message;
}

export function sessionEndedError(): AppError {
  return new AppError('UNAUTHENTICATED', 'Сессия завершена. Войдите снова.');
}
