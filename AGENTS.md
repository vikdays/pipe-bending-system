# Repository instructions

This is a monorepo for an offline pipe bending control prototype.

- `apps/mobile`: existing React Native + TypeScript + Expo application. Read `apps/mobile/AGENTS.md` before changing it. Resolve its package versions, assets and Expo commands from that directory.
- `apps/desktop`: Windows Wails + Go application with a React + TypeScript frontend. Authentication, the pipe register, certificate references, Excel import drafts, machines, configurations and tasks use local SQLite. PDF and imported DOCX originals are private content-addressed attachments. The technical-card importer supports the two studied GT532/GT1022 DOCX layouts; source cells and footnotes are retained, imported configurations require explicit source review before task preparation. Imports require preview and explicit selection; conflicts never overwrite pipes. Tasks have a persisted execution order. Drafts are editable; preparation validates source revisions, freezes pipe/configuration/machine snapshots and makes the task immutable. Batch creation is atomic and retry-safe. macOS is used for development; Windows runtime verification is still required.
- `packages/domain` and `packages/contracts`: future platform-independent TypeScript code, not yet configured as npm packages.
- `docs`: requirements, design decisions and development instructions.
- `examples`: fictional data only.

Preserve the existing root Git history. Do not create nested `.git` directories or generate a new mobile project over the existing code.

There is no npm workspace setup yet. Each application keeps its own dependency installation and lock-file. The root package only forwards commands.

Run `npm run typecheck` and `npm run lint` from the root before declaring work done. Report missing tooling honestly. Do not install dependencies after a denied installation request without renewed authorization.

Keep Expo Router routes in `apps/mobile/src/app`; components, hooks and application logic belong outside the route directory. Follow the versioned Expo documentation requirements in the mobile instructions. Do not manually create generated `apps/mobile/ios` or `apps/mobile/android` directories.

Each device owns its SQLite database. Exchange versioned task/result files and snapshots, never replace a device database with the other device's database. Resume the same session and repeat measurements within the same physical bend. Only explicitly confirmed measurements enter the protocol.

Start with a simulator. Do not invent sensor transports, production formulas, tolerances, operating modes or the method for obtaining H. The application does not control the machine. Windows and Android are prototype targets; RED OS is future work.

Keep real production documents and runtime data outside tracked source, such as in `local-data`. Do not upload them to GitHub. Empty scaffold directories use `.gitkeep`; remove the marker when adding real code if convenient.
