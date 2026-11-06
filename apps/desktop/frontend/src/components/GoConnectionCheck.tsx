import { useState } from 'react';
import { GreetService } from '../../bindings/changeme';

export function GoConnectionCheck() {
  const [message, setMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  async function handleCheck() {
    setIsLoading(true);
    setMessage('');

    try {
      const response = await GreetService.Greet('Виктория');
      setMessage(response);
    } catch (error) {
      const details =
        error instanceof Error ? error.message : String(error);

      setMessage(`Не удалось вызвать Go: ${details}`);
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="demo-banner">
      <button
        type="button"
        className="button button--primary"
        onClick={handleCheck}
        disabled={isLoading}
      >
        {isLoading ? 'Проверяем…' : 'Проверить связь с Go'}
      </button>

      <p role="status">{message}</p>
    </div>
  );
}