# devport

<p align="right">
  <a href="README.md">English</a>
</p>

로컬 개발용 포트 충돌을 줄이기 위한 CLI입니다.

`devport`는 프로젝트별로 안정적인 포트를 배정하고, 감지된 프레임워크에 맞는 env 파일을 작성하며, `PORT`가 주입된 상태로 하위 프로세스를 실행할 수 있게 해줍니다.

README의 포트 번호는 예시입니다. 실제 번호는 해당 프레임워크 범위에서 비어 있는 포트에 따라 달라집니다.

## 설치

```bash
# npm
npm install -g @vp-k/devport

# Go
go install github.com/user01/devport@latest
```

## 빠른 시작

```bash
# 1. 현재 프로젝트 포트 배정
cd my-app
devport get
# 3000

# 2. 감지된 프레임워크에 맞는 env 파일 작성
devport env
# Wrote PORT=3000 to .env.local

# 3. 현재 등록 상태 확인
devport status --json

# 4. PORT를 주입해서 개발 서버 실행
devport exec -- npm run dev
```

처음을 한 번에 설정하려면:

```bash
devport init --yes
```

`init`은 포트를 배정하고 env 파일을 작성하며, 필요하면 `package.json`에 `"predev": "devport env"`를 추가하고 `.env.local`을 `.gitignore`에 넣습니다.

## 명령어

| 명령어 | 설명 |
|---|---|
| `devport get` | 현재 프로젝트 포트를 조회하거나 새로 배정 |
| `devport env` | 감지된 env 파일에 포트 변수 기록 |
| `devport init` | 포트 배정, env 파일 작성, 기본 설정까지 한 번에 수행 |
| `devport exec -- <command>` | `PORT`를 주입한 상태로 하위 프로세스 실행 |
| `devport list` | 모든 등록 항목 출력 |
| `devport status` | 현재 프로젝트의 포트와 상태 출력 |
| `devport free [key|port]` | 특정 등록 해제, `--all`로 전체 해제 가능 |
| `devport reset [key]` | 새 포트로 강제 재배정 |
| `devport clean` | 오래됐거나 잘못된 등록 정리 |
| `devport doctor` | 레지스트리 진단 및 일부 자동 수정 |
| `devport export` | JSON 또는 CSV로 내보내기 |
| `devport import <file>` | JSON export 파일 가져오기 |

### 자주 쓰는 예시

```bash
devport get --json
devport get --range-min 4000 --range-max 4999
devport get --framework express
```

```bash
devport env
devport env --output .env.custom
devport env --var-name MY_PORT
devport env --framework vite
```

```bash
devport exec -- npm run dev
devport exec -- node server.js
devport exec --auto-free -- npm start
```

```bash
devport list --json
devport status --json
devport doctor --fix
```

`doctor --fix`는 안전한 수정만 수행합니다. 중복 포트 항목이 있으면 `allocatedAt`이 가장 최신인 항목을 남기고, 더 오래된 중복 항목을 제거합니다.

## 프레임워크 감지

프레임워크 감지는 기본 env 파일, 환경 변수명, 포트 범위를 결정합니다.

감지 우선순위:

1. `next.config.*`, `vite.config.*`, `angular.json`, `wrangler.toml`, `nuxt.config.*`, `svelte.config.*`, `remix.config.js`
2. `bun.lockb`, `bunfig.toml`, `deno.json`, `deno.jsonc`
3. `go.mod`
4. `package.json`의 `dependencies`, `devDependencies`

| 프레임워크 | 감지 기준 | Env 파일 | 변수명 | 포트 범위 |
|---|---|---|---|---|
| Next.js | `next.config.*` 또는 `next` 의존성 | `.env.local` | `PORT` | 3000-3999 |
| Vite | `vite.config.*` 또는 `vite` 의존성 | `.env.local` | `VITE_PORT` | 5000-5999 |
| Express | `express` 의존성 | `.env` | `PORT` | 4000-4999 |
| NestJS | `@nestjs/core` 의존성 | `.env` | `PORT` | 3000-3999 |
| CRA | `react-scripts` 의존성 | `.env.local` | `PORT` | 3000-3999 |
| Angular | `angular.json` | `.env.local` | `PORT` | 4200-4299 |
| Nuxt | `nuxt.config.*` | `.env` | `PORT` | 3000-3999 |
| Remix | `remix.config.js` 또는 `@remix-run/dev` | `.env` | `PORT` | 3000-3999 |
| SvelteKit | `svelte.config.*` | `.env` | `PORT` | 5000-5999 |
| Cloudflare Workers | `wrangler.toml` | `.dev.vars` | `PORT` | 8787-8799 |
| Bun | `bun.lockb` 또는 `bunfig.toml` | `.env` | `PORT` | 3000-3999 |
| Deno | `deno.json` 또는 `deno.jsonc` | `.env` | `PORT` | 3000-3999 |
| Fastify | `fastify` 의존성 | `.env` | `PORT` | 3000-3999 |
| Hono (Node) | `hono`와 `@hono/node-server` | `.env` | `PORT` | 3000-3999 |
| Go, Gin, Echo, Fiber, Chi | `go.mod` | `.env` | `PORT` | 8000-8999 |

프레임워크를 감지하지 못하면 `devport env`와 `devport status`는 `.env.local`과 `PORT`를 기본값으로 사용하고, 포트 배정 범위는 `3000-9999`로 떨어집니다.

## Windows

CMD나 PowerShell에서는 `$(devport get)` 방식이 바로 동작하지 않습니다. 아래 패턴을 권장합니다.

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

## 동작 방식

- 레지스트리는 `~/.devports.json`에 저장됩니다.
- 쓰기 작업은 `~/.devports.json.lock` 파일 잠금으로 보호됩니다.
- 저장 시 임시 파일과 atomic rename을 사용해 손상 가능성을 줄입니다.
- 프로젝트 키는 `package.json` 이름, git remote 해시, 경로 해시 순서로 결정됩니다.
- 한 번 배정된 포트는 `devport reset` 또는 `devport free` 전까지 유지됩니다.

## 레지스트리

예시:

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

`devport export`가 만든 JSON 배열은 `devport import`로 다시 가져올 수 있습니다.

## 저장소 검증

저장소에는 README 흐름을 점검하는 스모크 테스트가 포함되어 있으며 다음 경로를 확인합니다.

- `get`
- `env`
- `status --json`
- `list --json`
- `exec`

npm 플랫폼 패키지를 배포하기 전에는 `go run ./scripts/stage_npm_binaries.go`를 실행해서 각 `npm/platforms/*` 패키지의 `bin/` 아래에 실제 바이너리를 채워야 합니다.

## 문제 해결

### 포트가 배정됐는데 서버 기동 시 바인딩 실패 (Windows WSL / Hyper-V)

`devport get`은 배정 시점에 포트를 바인딩할 수 있는지 확인하지만, Windows는 Hyper-V나 WSL을 통해 포트 범위를 예약해 두면서 실제로 리스닝하지 않는 경우가 있습니다. 배정 시점에는 통과했더라도 개발 서버가 실제로 바인딩할 때 실패할 수 있습니다.

**해결 순서:**

1. `devport reset`으로 재배정합니다. 같은 범위에서 새 포트를 다시 찾습니다.
2. 같은 문제가 반복되면 Windows의 예약 범위를 확인합니다.
   ```powershell
   netsh int ipv4 show excludedportrange protocol=tcp
   ```
3. 예약 범위가 해당 프레임워크 범위 전체를 덮고 있으면, 현재 등록을 해제하고 다른 범위로 재배정합니다.
   ```bash
   devport free --force
   devport get --range-min 10000 --range-max 10999
   ```
   또는 다른 프레임워크 플래그로 기본 범위를 바꿉니다.
   ```bash
   devport free --force
   devport get --framework go   # 8000-8999 범위 사용
   ```

### macOS에서 포트 5000 또는 7000을 사용할 수 없음 (AirPlay 수신기)

macOS Monterey 이후 버전은 AirPlay 수신기가 포트 5000(경우에 따라 7000)을 사용합니다. `devport`가 이 포트를 배정했는데 서버 기동에 실패하면, **시스템 설정 → 일반 → AirDrop 및 Handoff**에서 AirPlay 수신기를 비활성화하거나 다른 범위를 지정합니다.

```bash
devport free --force
devport get --range-min 5100 --range-max 5999
```

## 라이선스

MIT
