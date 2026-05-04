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

Source pages live in **`docs/docs/`**.

## GitHub Pages

The workflow **[`.github/workflows/deploy-docs.yml`](../.github/workflows/deploy-docs.yml)** builds with Bun and deploys via the official **GitHub Actions → Pages** flow.

1. Repo **Settings → Pages**: set **Build and deployment → Source** to **GitHub Actions** (not “Deploy from a branch”).
2. Push to **`main`** with changes under **`docs/`**, or run the workflow manually (**Actions → Deploy docs → Run workflow**).

CI sets **`DOCUSAURUS_URL`** and **`DOCUSAURUS_BASE_URL`** from **`github.repository_owner`** and **`github.event.repository.name`**, so project-site URLs look like **`https://<owner>.github.io/<repo>/`**.

Local preview keeps **`baseUrl` `/`** unless you mimic production:

```bash
DOCUSAURUS_URL=https://your-user.github.io DOCUSAURUS_BASE_URL=/your-repo/ bun run build && bun run serve
```
