import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { Modal } from '../../components/Modal';
import { authClient, authErrorMessage } from './authClient';
import type { AuthUser } from './authClientCore';

export function AccountModal({
  user,
  onClose,
}: {
  user: AuthUser;
  onClose: () => void;
}) {
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(user.role === 'admin');

  useEffect(() => {
    if (user.role !== 'admin') return;
    let active = true;
    void authClient
      .listUsers()
      .then((result) => {
        if (active) setUsers(result);
      })
      .catch((error) => {
        if (active) setError(authErrorMessage(error));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [user.role]);

  async function createUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await authClient.createUser(
        String(data.get('username')),
        String(data.get('password')),
        String(data.get('role')),
      );
      form.reset();
      setUsers(await authClient.listUsers());
      setNotice('Учётная запись создана на этом компьютере.');
    } catch (error) {
      setError(authErrorMessage(error));
    } finally {
      const password = form.elements.namedItem('password');
      if (password instanceof HTMLInputElement) password.value = '';
      setBusy(false);
    }
  }

  async function changePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    if (data.get('newPassword') !== data.get('confirmPassword')) {
      setError('Новые пароли не совпадают.');
      return;
    }
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await authClient.changePassword(
        String(data.get('currentPassword')),
        String(data.get('newPassword')),
      );
      // All sessions, including this one, are revoked by Go; App returns to sign-in.
    } catch (error) {
      setError(authErrorMessage(error));
    } finally {
      form.reset();
      setBusy(false);
    }
  }

  async function toggleAccount(account: AuthUser) {
    if (busy) return;
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await authClient.setUserEnabled(account.id, !account.enabled);
      setUsers(await authClient.listUsers());
      setNotice(
        account.enabled
          ? 'Учётная запись заблокирована, её сессии завершены.'
          : 'Учётная запись разблокирована.',
      );
    } catch (error) {
      setError(authErrorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      title="Учётная запись"
      description={`${user.username} · ${user.role === 'admin' ? 'Руководитель работ' : 'Машинист'}`}
      onClose={onClose}
      closeDisabled={busy}
    >
      <div className="account-sections">
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        {notice && (
          <p className="auth-notice" role="status">
            {notice}
          </p>
        )}
        <form className="task-form" onSubmit={changePassword}>
          <h3>Сменить пароль</h3>
          <label className="form-field">
            Текущий пароль
            <input
              name="currentPassword"
              type="password"
              autoComplete="current-password"
              required
              maxLength={128}
              disabled={busy}
            />
          </label>
          <label className="form-field">
            Новый пароль
            <input
              name="newPassword"
              type="password"
              autoComplete="new-password"
              required
              minLength={15}
              maxLength={128}
              disabled={busy}
            />
          </label>
          <label className="form-field">
            Повторите новый пароль
            <input
              name="confirmPassword"
              type="password"
              autoComplete="new-password"
              required
              minLength={15}
              maxLength={128}
              disabled={busy}
            />
          </label>
          <p className="auth-help">
            От 15 до 128 символов. После смены пароля потребуется войти снова на
            всех открытых сеансах этого компьютера.
          </p>
          <button className="button button--secondary" disabled={busy}>
            Сменить пароль и выйти
          </button>
        </form>
        {user.role === 'admin' && (
          <>
            <section aria-label="Пользователи компьютера">
              <h3>Пользователи этого компьютера</h3>
              {loading ? (
                <p role="status">Загрузка…</p>
              ) : (
                <ul className="account-users">
                  {users.map((account) => (
                    <li key={account.id}>
                      <span>
                        <strong>{account.username}</strong>
                        <small>
                          {account.role === 'admin'
                            ? 'Руководитель работ'
                            : 'Машинист'}
                          {!account.enabled ? ' · Заблокирован' : ''}
                        </small>
                      </span>
                      <button
                        className="button button--secondary"
                        disabled={busy || account.id === user.id}
                        onClick={() => {
                          void toggleAccount(account);
                        }}
                      >
                        {account.enabled ? 'Заблокировать' : 'Разблокировать'}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </section>
            <form className="task-form" onSubmit={createUser}>
              <h3>Создать учётную запись</h3>
              <label className="form-field">
                Логин нового пользователя
                <input
                  name="username"
                  autoComplete="off"
                  autoCapitalize="none"
                  spellCheck={false}
                  required
                  pattern="[a-zA-Z][a-zA-Z0-9._\-]{2,63}"
                  minLength={3}
                  maxLength={64}
                  disabled={busy}
                />
              </label>
              <label className="form-field">
                Пароль нового пользователя
                <input
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={15}
                  maxLength={128}
                  disabled={busy}
                />
              </label>
              <label className="form-field">
                Роль
                <select name="role" defaultValue="operator" disabled={busy}>
                  <option value="operator">Машинист</option>
                  <option value="admin">Руководитель работ</option>
                </select>
              </label>
              <p className="auth-help">
                Учётные записи действуют только на этом компьютере и не
                передаются в файлах заданий.
              </p>
              <button className="button button--primary" disabled={busy}>
                Создать пользователя
              </button>
            </form>
          </>
        )}
      </div>
    </Modal>
  );
}
