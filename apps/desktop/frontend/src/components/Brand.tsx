export function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <div className={`brand ${compact ? 'brand--compact' : ''}`}>
      <span className="brand-mark" aria-hidden="true">
        <svg viewBox="0 0 40 40" fill="none">
          <path
            d="M9 11h8a12 12 0 0 1 12 12v6"
            stroke="currentColor"
            strokeWidth="8"
          />
          <path
            d="M6 6v10m18 16h10"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
          />
        </svg>
      </span>
      <div>
        <strong>Контроль гибки</strong>
        <span>Цифровая система</span>
      </div>
    </div>
  );
}
