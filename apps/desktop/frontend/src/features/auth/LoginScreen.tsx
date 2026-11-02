import { useState } from 'react';
import type { FormEvent } from 'react';
import { Brand } from '../../components/Brand';
import { Icon } from '../../components/Icon';
import { PipeBlueprint } from '../../components/PipeBlueprint';
import { authErrorMessage } from './authClient';

export function LoginScreen({
  onSignIn,
  setup = false,
  message,
  messageIsError = false,
}: {
  onSignIn: (
    username: string,
    password: string,
    setup: boolean,
  ) => Promise<void>;
  setup?: boolean;
  message?: string;
  messageIsError?: boolean;
}) {
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (isSubmitting) return;
    if (setup && password !== confirmPassword) {
      setError('Пароли не совпадают.');
      return;
    }
    setIsSubmitting(true);
    setError('');
    try {
      await onSignIn(login, password, setup);
    } catch (error) {
      setError(authErrorMessage(error));
    } finally {
      setPassword('');
      setConfirmPassword('');
      setIsSubmitting(false);
    }
  }

  return (
    <main className="login-screen">
      <section className="login-story" aria-label="О системе">
        <Brand />
        <div className="login-story-content">
          <h1>
            Цифровой контроль
            <br />
            гибки труб
          </h1>
          <p>
            Подготовка заданий, история измерений
            <br className="desktop-break" /> и результаты изготовления — в одной
            системе.
          </p>
          <PipeBlueprint />
          <div className="story-points">
            <span>
              <Icon name="clipboard" size={18} />
              Задания
            </span>
            <i />
            <span>
              <Icon name="pipe" size={18} />
              Измерения
            </span>
            <i />
            <span>
              <Icon name="file" size={18} />
              Протоколы
            </span>
          </div>
        </div>
        <div className="login-story-footer">
          <Icon name="shield" size={17} />
          <span>Автономная работа на предприятии</span>
          <span className="version-label">Прототип 0.1</span>
        </div>
      </section>

      <section className="login-form-section">
        <div className="login-form-wrap">
          <div className="workspace-icon">
            <Icon name="user" size={25} />
          </div>
          <span className="eyebrow">РАБОЧЕЕ МЕСТО РУКОВОДИТЕЛЯ</span>
          <h2>{setup ? 'Первый запуск' : 'Вход в систему'}</h2>
          <p className="login-description">
            {setup
              ? 'Создайте учётную запись руководителя для этого компьютера.'
              : 'Войдите, чтобы продолжить работу с заданиями и результатами.'}
          </p>
          {message && (
            <p
              className={messageIsError ? 'form-error' : 'auth-notice'}
              role={messageIsError ? 'alert' : 'status'}
            >
              {message}
            </p>
          )}
          <form onSubmit={handleSubmit} className="login-form">
            <label className="field-label" htmlFor="login">
              Логин
            </label>
            <div className="input-with-icon">
              <Icon name="user" size={19} />
              <input
                id="login"
                name="username"
                autoComplete="username"
                value={login}
                required
                minLength={3}
                maxLength={64}
                disabled={isSubmitting}
                autoCapitalize="none"
                spellCheck={false}
                onChange={(event) => {
                  setLogin(event.target.value);
                  setError('');
                }}
                placeholder="Введите логин"
              />
            </div>
            <label className="field-label password-label" htmlFor="password">
              Пароль
            </label>
            <div
              className={`input-with-icon ${error ? 'input-with-icon--error' : ''}`}
            >
              <Icon name="lock" size={19} />
              <input
                id="password"
                name="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete={setup ? 'new-password' : 'current-password'}
                value={password}
                required
                minLength={setup ? 15 : undefined}
                maxLength={128}
                disabled={isSubmitting}
                aria-invalid={Boolean(error)}
                aria-describedby={error ? 'login-error' : undefined}
                onChange={(event) => {
                  setPassword(event.target.value);
                  setError('');
                }}
                placeholder="Введите пароль"
              />
              <button
                type="button"
                className="password-toggle"
                aria-label={showPassword ? 'Скрыть пароль' : 'Показать пароль'}
                aria-pressed={showPassword}
                onClick={() => setShowPassword(!showPassword)}
              >
                <Icon name={showPassword ? 'eyeOff' : 'eye'} size={19} />
              </button>
            </div>
            {setup && (
              <>
                <p className="auth-help">
                  Логин: 3–64 символа латиницей, начиная с буквы. Пароль: 15–128
                  символов, например длинная уникальная фраза.
                </p>
                <label className="field-label" htmlFor="confirm-password">
                  Повторите пароль
                </label>
                <div className="input-with-icon">
                  <Icon name="lock" size={19} />
                  <input
                    id="confirm-password"
                    type="password"
                    autoComplete="new-password"
                    required
                    minLength={15}
                    maxLength={128}
                    disabled={isSubmitting}
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                  />
                </div>
              </>
            )}
            {error && (
              <p className="form-error" id="login-error" role="alert">
                {error}
              </p>
            )}
            <button
              className="button button--primary login-submit"
              type="submit"
              disabled={isSubmitting}
            >
              {isSubmitting
                ? 'Подождите…'
                : setup
                  ? 'Создать учётную запись'
                  : 'Войти'}
              <Icon name="arrow" size={19} />
            </button>
          </form>
          <div className="login-account-note">
            <Icon name="shield" size={19} />
            <div>
              <strong>Локальная учётная запись</strong>
              <p>
                Для входа Интернет не требуется. После закрытия приложения
                войдите снова.
              </p>
            </div>
          </div>
        </div>
        <footer className="login-form-footer">
          Система контроля изготовления отводов холодного гнутья
        </footer>
      </section>
    </main>
  );
}
