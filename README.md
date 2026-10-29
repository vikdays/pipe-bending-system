# Цифровой контроль гибки труб

Монорепозиторий ВКР «Разработка информационной системы цифрового контроля процесса изготовления отводов холодного гнутья».

Один репозиторий содержит приложение руководителя для Windows, приложение машиниста для Android и общие материалы разработки.

## Карта проекта

```text
pipe-bending-mobile/             ← корень монорепозитория, один .git
├── apps/
│   ├── mobile/                 ← Android: React Native + Expo
│   │   ├── src/app/            ← экраны и навигация Expo Router
│   │   ├── src/components/     ← переиспользуемые мобильные компоненты
│   │   ├── src/features/       ← задания, сеансы, архив
│   │   ├── src/services/       ← источники показаний, сначала симулятор
│   │   ├── src/storage/        ← SQLite и миграции планшета
│   │   ├── src/exchange/       ← выбор файлов, импорт и экспорт на Android
│   │   ├── src/reports/        ← формирование протокола на планшете
│   │   ├── assets/             ← изображения и другие ресурсы
│   │   ├── package.json
│   │   └── package-lock.json   ← сохранённый lock-файл мобильного приложения
│   └── desktop/                ← Windows: Wails + React + Go
│       ├── frontend/src/       ← интерфейс React
│       └── internal/           ← Go: SQLite и файловый обмен
├── packages/
│   ├── domain/src/             ← будущие общие предметные типы и правила
│   └── contracts/src/          ← будущие форматы задания/результата и проверки
├── examples/                   ← только вымышленные задания, результаты, конфигурации
├── docs/                       ← контекст, архитектура, решения и проверки
├── tests/integration/          ← будущие проверки обмена между приложениями
├── scripts/                    ← будущие вспомогательные команды
├── local-data/                 ← локальные материалы, содержимое исключено из Git
└── package.json                ← удобные команды из корня
```

## Запуск мобильного приложения

```bash
cd /Users/vikdays/Documents/devtools/projects/pipe-bending-mobile
npm run mobile:start
```


```bash
npm run mobile:start -- --clear
```


```bash
cd /Users/vikdays/Documents/devtools/projects/pipe-bending-mobile/apps/mobile
npx expo start --go
```