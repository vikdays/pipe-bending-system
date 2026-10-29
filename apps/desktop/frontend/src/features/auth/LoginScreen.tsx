import { useState } from 'react';
import type { FormEvent } from 'react';
import { Brand } from '../../components/Brand';
import { Icon } from '../../components/Icon';
import { PipeBlueprint } from '../../components/PipeBlueprint';

export function LoginScreen({ onSignIn }: { onSignIn: () => void }) {
  const [login, setLogin] = useState('admin');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (login.trim() !== 'admin' || password !== 'demo2026') {
      setError(
        'Неверный логин или пароль. Для демовхода используйте данные ниже.',
      );
      return;
    }
    onSignIn();
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
          <h2>Вход в систему</h2>
          <p className="login-description">
            Войдите, чтобы продолжить работу
            <br />с заданиями и результатами.
          </p>
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
                autoComplete="current-password"
                value={password}
                required
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
            {error && (
              <p className="form-error" id="login-error" role="alert">
                {error}
              </p>
            )}
            <button
              className="button button--primary login-submit"
              type="submit"
            >
              Войти
              <Icon name="arrow" size={19} />
            </button>
          </form>
          <div className="demo-access">
            <Icon name="info" size={19} />
            <div>
              <strong>Демонстрационный доступ</strong>
              <p>
                Логин: <code>admin</code>
                <span>·</span>Пароль: <code>demo2026</code>
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
