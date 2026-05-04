# Sash documentation

Static site for **[Sash](https://github.com/zackarysantana/sash)** ([Docusaurus](https://docusaurus.io/), Bun).

## Commands

From the **repository root** (via [included Taskfile](../Taskfile.yml)):

```bash
task docs:install
task docs:dev
task docs:build
```

Directly in **`docs/`**:

```bash
bun install
bun start
bun run build
bun run serve
```

Source pages live in **`docs/docs/`**. Adjust **`docusaurus.config.ts`** `url` / `baseUrl` when you deploy.
