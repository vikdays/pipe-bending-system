import { Icon } from './Icon';

export function Toast({
  message,
  onDismiss,
}: {
  message: string;
  onDismiss: () => void;
}) {
  return (
    <div className="toast" role="status">
      <span>
        <Icon name="check" size={19} />
      </span>
      {message}
      <button
        className="icon-button"
        aria-label="Закрыть уведомление"
        onClick={onDismiss}
      >
        <Icon name="close" size={16} />
      </button>
    </div>
  );
}
