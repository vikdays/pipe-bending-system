# Настольное приложение руководителя

Цель: Windows, Wails + Go, React + TypeScript. Wails-проект, зависимости и база ещё не созданы.

- `frontend/src/components` — общие компоненты настольного интерфейса.
- `frontend/src/features/tasks` — подготовка задания и шапки протокола.
- `frontend/src/features/pipes` — реестр физических труб и связи с документами.
- `frontend/src/features/archive` — импортированные результаты и просмотр протоколов.
- `internal/storage/migrations` — миграции SQLite ПК; рядом будет код доступа к БД.
- `internal/exchange` — Go-код чтения и записи файлов заданий/результатов.
