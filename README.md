# devport

<p align="right">
  <a href="README.ko.md">Korean</a>
</p>

Conflict-free port allocation for local development.

`devport` gives each project a stable local port, writes the right env file for the detected framework, and can run child processes with the allocated port injected.

Examples in this README use sample port numbers. The actual port depends on what is free inside the framework range.

## Installation

```bash
# npm
npm install -g @vp-k/devport

# Go
go install github.com/vp-k/devport@latest
```

## Quick Start

```bash
# 1. Allocate a stable port for the current project
cd my-app
devport get
# 3000

# 2. Write the detected env file
devport env
# Wrote PORT=3000 to .env.local

# 3. Check the current registration
devport status --json

# 4. Run a command with the allocated port injected
devport exec -- npm run dev
```

Or do the initial setup in one command:

```bash
devport init --yes
```

`init` allocates a port, writes the env file, optionally adds `"predev": "devport env"` to `package.json`, and can add `.env.local` to `.gitignore`.

## Commands

| Command | Purpose |
|---|---|
| `devport get` | Get or allocate the current project's port |
| `devport env` | Write the detected env file with the correct port variable |
| `devport init` | Allocate a port, write the env file, and offer common project setup changes |
| `devport exec -- <command>` | Run a child process with the allocated port injected |
| `devport list` | List every registered project |
| `devport status` | Show the current project's port and status |
| `devport free [key|port]` | Remove one registration, or use `--all` to clear everything |
| `devport reset [key]` | Force a new port allocation |
| `devport clean` | Remove stale, old, or all registrations |
| `devport doctor` | Diagnose the registry and optionally fix common problems |
| `devport export` | Export registrations as JSON or CSV |
| `devport import <file>` | Import registrations from a JSON export |

### Common Flags

`devport get`

```bash
devport get --json
devport get --range-min 4000 --range-max 4999
devport get --framework express
```

`devport env`

```bash
devport env
devport env --output .env.custom
devport env --var-name MY_PORT
devport env --framework vite
```

`devport exec`

```bash
devport exec -- npm run dev
devport exec -- node server.js
devport exec --auto-free -- npm start
```

`devport list`

```bash
devport list
devport list --verbose
devport list --json
```

`devport doctor`

```bash
devport doctor
devport doctor --fix
```

`doctor --fix` performs safe repairs only. For duplicate port assignments it keeps the newest entry by `allocatedAt` and removes older duplicates.

`devport export` and `devport import`

```bash
devport export --format json
devport export --format csv --output backup.csv
devport import backup.json
devport import backup.json --overwrite
devport import backup.json --dry-run
```

## Framework Detection

Framework detection controls the default env file, env variable name, and allocation range.

Detection priority:

1. Config files such as `next.config.*`, `vite.config.*`, `angular.json`, `wrangler.toml`, `nuxt.config.*`, `svelte.config.*`, and `remix.config.js`
2. Runtime files such as `bun.lockb`, `bunfig.toml`, `deno.json`, and `deno.jsonc`
3. `go.mod`
4. `package.json` dependencies and devDependencies

| Framework | Detects from | Env file | Variable | Port range |
|---|---|---|---|---|
| Next.js | `next.config.*` or `next` dependency | `.env.local` | `PORT` | 3000-3999 |
| Vite | `vite.config.*` or `vite` dependency | `.env.local` | `VITE_PORT` | 5000-5999 |
| Express | `express` dependency | `.env` | `PORT` | 4000-4999 |
| NestJS | `@nestjs/core` dependency | `.env` | `PORT` | 3000-3999 |
| CRA | `react-scripts` dependency | `.env.local` | `PORT` | 3000-3999 |
| Angular | `angular.json` | `.env.local` | `PORT` | 4200-4299 |
| Nuxt | `nuxt.config.*` | `.env` | `PORT` | 3000-3999 |
| Remix | `remix.config.js` or `@remix-run/dev` | `.env` | `PORT` | 3000-3999 |
| SvelteKit | `svelte.config.*` | `.env` | `PORT` | 5000-5999 |
| Cloudflare Workers | `wrangler.toml` | `.dev.vars` | `PORT` | 8787-8799 |
| Bun | `bun.lockb` or `bunfig.toml` | `.env` | `PORT` | 3000-3999 |
| Deno | `deno.json` or `deno.jsonc` | `.env` | `PORT` | 3000-3999 |
| Fastify | `fastify` dependency | `.env` | `PORT` | 3000-3999 |
| Hono (Node) | `hono` and `@hono/node-server` | `.env` | `PORT` | 3000-3999 |
| Go, Gin, Echo, Fiber, Chi | `go.mod` | `.env` | `PORT` | 8000-8999 |

If the framework is unknown, `devport env` and `devport status` fall back to `.env.local` with `PORT`, and allocation falls back to the default `3000-9999` range.

## Windows

Shell substitution such as `$(devport get)` does not work in CMD or PowerShell. Use one of these patterns instead:

```powershell
devport init --yes
```

```jsonc
{
  "scripts": {
    "predev": "devport env",
    "dev": "node server.js"
  }
}
```

```powershell
devport exec -- node server.js
```

## How It Works

- The registry is stored in `~/.devports.json`.
- Writes are protected by a lock file at `~/.devports.json.lock`.
- Registry updates use a temp file plus atomic rename to reduce corruption risk.
- Project identity is resolved in this order: `package.json` name, git remote hash, path hash.
- Once a project gets a port, it keeps that port until `devport reset` or `devport free`.

## Registry

Example registry file:

```json
{
  "version": 1,
  "meta": {
    "createdAt": "2026-01-01T00:00:00Z",
    "updatedAt": "2026-01-01T12:00:00Z"
  },
  "entries": {
    "my-app": {
      "port": 3000,
      "keySource": "package.json",
      "displayName": "my-app",
      "projectPath": "/Users/alice/work/my-app",
      "framework": "next",
      "allocatedAt": "2026-01-01T00:00:00Z",
      "lastAccessedAt": "2026-01-01T12:00:00Z"
    }
  }
}
```

`devport export` writes a JSON array that can be imported back with `devport import`.

## Development Check

The repository includes command-level tests and a README smoke test that covers:

- `get`
- `env`
- `status --json`
- `list --json`
- `exec`

Before publishing the npm platform packages, run `go run ./scripts/stage_npm_binaries.go` so each `npm/platforms/*` package contains the correct binary in `bin/`.

## Troubleshooting

### Port allocated but server fails to bind (Windows WSL / Hyper-V)

`devport get` checks whether a port can be bound at allocation time, but Windows can reserve port ranges through Hyper-V or WSL without actively listening on them. A port may pass the allocation check and still fail when your dev server tries to use it.

**Steps to resolve:**

1. Try `devport reset` — this re-probes and picks a new port from the same range.
2. If the problem repeats, check which ranges Windows has reserved:
   ```powershell
   netsh int ipv4 show excludedportrange protocol=tcp
   ```
3. If the reserved range covers the entire framework range, free the current registration and re-allocate with a custom range:
   ```bash
   devport free --force
   devport get --range-min 10000 --range-max 10999
   ```
   Or target a different default range using a framework override:
   ```bash
   devport free --force
   devport get --framework go   # targets 8000-8999
   ```

### Port 5000 or 7000 unavailable on macOS (AirPlay Receiver)

macOS Monterey and later run an AirPlay Receiver on port 5000 (and sometimes 7000). If `devport` allocates one of these ports and your server fails to start, either disable AirPlay Receiver in **System Settings → General → AirDrop & Handoff** or re-allocate with a custom range:

```bash
devport free --force
devport get --range-min 5100 --range-max 5999
```

## License

MIT
