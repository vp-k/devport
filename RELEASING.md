# Release Guide

## Prerequisites

```powershell
npm whoami                        # vp-k 확인
gh auth status                    # GitHub 인증 확인
goreleaser --version              # goreleaser 설치 확인
```

`~/.npmrc`에 아래 설정이 있어야 합니다:
```
//registry.npmjs.org/:_authToken=<granular token, 2FA bypass + read/write>
```

---

## 1. 버전 번호 업데이트

`npm/package.json`과 플랫폼 패키지 5개의 `version` 필드를 동일하게 수정합니다.

```
npm/package.json
npm/platforms/devport-darwin-arm64/package.json
npm/platforms/devport-darwin-x64/package.json
npm/platforms/devport-linux-arm64/package.json
npm/platforms/devport-linux-x64/package.json
npm/platforms/devport-win32-x64/package.json
```

## 2. go mod tidy 및 커밋

goreleaser의 `before.hooks`가 `go mod tidy`를 실행하므로 **미리** 로컬에서 실행해 커밋에 포함시킵니다. 그렇지 않으면 goreleaser가 dirty state로 실패합니다.

```powershell
go mod tidy
git add <변경 파일들> go.mod go.sum
git commit -m "release: vX.Y.Z"
git push origin HEAD
```

## 3. npm 플랫폼 패키지 publish (5개)

root 패키지의 `optionalDependencies`가 플랫폼 패키지를 참조하므로 **반드시 먼저** publish합니다.

```powershell
cd npm/platforms/devport-darwin-arm64;  npm.cmd publish --access public
cd ../devport-darwin-x64;               npm.cmd publish --access public
cd ../devport-linux-arm64;              npm.cmd publish --access public
cd ../devport-linux-x64;               npm.cmd publish --access public
cd ../devport-win32-x64;                npm.cmd publish --access public
```

## 4. npm root 패키지 publish

```powershell
cd ../../
npm.cmd publish --access public
```

## 5. GitHub Release (goreleaser)

```powershell
$env:GITHUB_TOKEN = (gh auth token)
git tag vX.Y.Z
git push origin vX.Y.Z
goreleaser release --clean
```

---

## Troubleshooting

| 증상 | 원인 | 해결 |
|---|---|---|
| `E403 Two-factor authentication required` | npm token에 2FA bypass 미설정 | Granular token 재발급 시 "bypass 2FA" 체크 |
| `ENOVERSIONS` on install | `~/.npmrc`의 `min-release-age=7` 설정 | `npm install -g @vp-k/devport --min-release-age=0` |
| `git is in a dirty state` | goreleaser 실행 전 uncommitted 파일 존재 | `git status` 확인 후 커밋 또는 `.gitignore` 추가 |
| `git tag vX.Y.Z was not made against commit ...` | 태그 생성 후 추가 커밋 발생 | 아래 태그 재이동 절차 참고 |

### 태그 재이동

```powershell
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z
git tag vX.Y.Z
git push origin vX.Y.Z
```
