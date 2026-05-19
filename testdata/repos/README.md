# Fixture Corpus

The repositories in this directory are reduced, reviewable fixtures derived from public project layouts.

- Keep only files that affect detector behavior, provider resolution, or managed-output contracts.
- Remove history, secrets, vendored dependencies, build outputs, and unrelated source files.
- Prefer small fixtures that still preserve the original repository shape that matters for the test.

## Fixtures

### `next-vscode-app`

- Derived from a public Next.js app layout with checked-in editor metadata.
- Preserves `package.json`, `next.config.js`, `app/`, `.vscode/`, and `.idea/` signals.
- Exercises `node`, `react`, `nextjs`, `visualstudiocode`, and `jetbrains` detection on one realistic repository shape.

### `laravel-jetbrains-app`

- Derived from a public Laravel app layout with checked-in JetBrains metadata.
- Preserves `composer.json`, `artisan`, and `.idea/` signals.
- Exercises `composer`, `laravel`, and `jetbrains` detection on a realistic PHP repository shape.
