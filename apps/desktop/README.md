# Настольное приложение руководителя

Цель: Windows, Wails + Go, React + TypeScript. Создан React-интерфейс: экран входа и главный экран руководителя. Нативная оболочка Wails и SQLite пока не подключены.

Запуск из корня монорепозитория:

```bash
npm run desktop:start
```

Откройте `http://127.0.0.1:5173`. Демонстрационный вход: `admin` / `demo2026`.
Настоящая проверка пароля в Go ещё не реализована. Все записи вымышленные;
созданные в интерфейсе черновики исчезают после обновления страницы.

Если зависимости ещё не установлены, выполните `npm ci` в `apps/desktop/frontend`.

Проверки из корня: `npm run desktop:typecheck`, `npm run desktop:lint`, `npm run desktop:test`, `npm run desktop:build`.
Результат сборки интерфейса — `frontend/dist`; это ещё не установщик Windows.

Основные файлы:

- `frontend/src/features/auth/LoginScreen.tsx` — форма входа и её состояния.
- `frontend/src/features/dashboard/DashboardScreen.tsx` — сборка рабочего стола из компонентов.
- `frontend/src/features/dashboard/useDashboard.ts` — состояние представления, загрузка, фильтрация и диалоги.
- `frontend/src/features/tasks` — таблица, фильтры, карточка и форма черновика; общие типы и демонстрационные данные.
- `frontend/src/features/tasks/taskService.ts` — асинхронный интерфейс получения и создания заданий, пока с хранением в памяти. Каждый новый вход создаёт независимый список; SQLite и Go ещё не подключены.
- `frontend/src/components/layout` — оболочка приложения и боковая навигация.
- `frontend/src/components` — логотип, иконки и диалоговое окно.
- `frontend/src/styles.css` — оформление обоих экранов.
- `frontend/tests/taskService.test.mjs` — проверки создания, отклонения некорректного ввода и изоляции данных сервиса. Выполняются встроенным тестовым инструментом Node.js 24.

- `frontend/src/components` — общие компоненты настольного интерфейса.
- `frontend/src/features/tasks` — подготовка задания и шапки протокола.
- `frontend/src/features/pipes` — реестр физических труб и связи с документами.
- `frontend/src/features/archive` — импортированные результаты и просмотр протоколов.
- `internal/storage/migrations` — миграции SQLite ПК; рядом будет код доступа к БД.
- `internal/exchange` — Go-код чтения и записи файлов заданий/результатов.
