# Build Instructions

## Prerequisites
- **Build Tool**: Go **1.25+**（go.mod の go ディレクティブ）。`go version` で確認。
- **Dependencies**: `go.mod`/`go.sum` で固定（`gopkg.in/yaml.v3`, `pgregory.net/rapid`[test], `charmbracelet/bubbletea`・`bubbles`・`lipgloss`, `golang.org/x/term`）。
- **External**: `git` 実行ファイル（git ツール利用時）。LM Studio は**ビルド/自動テストには不要**（実行時のみ）。
- **System**: macOS / Linux（ターミナルのプロセスグループ kill は Unix 前提）。ディスク数百MB（モジュールキャッシュ）。

## Build Steps

### 1. Install Dependencies
```bash
go mod download
```

### 2. Build All Packages
```bash
go build ./...
# 単一バイナリ生成:
make build          # -> bin/shiroutocode
# または
go build -o bin/shiroutocode ./cmd/shiroutocode
```

### 3. Cross-compile（配布用・任意）
```bash
make cross          # bin/shiroutocode-{darwin,linux}-{amd64,arm64}
```

### 4. Verify Build Success
- **Expected Output**: エラーなく終了（`go build ./...` は無出力）。
- **Build Artifacts**: `bin/shiroutocode`（単一静的バイナリ, 約10MB）。
- **Common Warnings**: なし（`go vet ./...` クリーン）。

## Troubleshooting
### Dependency Errors
- **Cause**: ネットワーク不通 / プロキシ。
- **Solution**: `GOPROXY` 設定、`go mod download` 再実行、`go clean -modcache`。

### Compilation Errors（古い Go）
- **Cause**: Go < 1.25。
- **Solution**: Go 1.25+ を導入（`log/slog` ほか使用）。

### `git`/`sh` 未検出（実行時）
- git ツールは `git` を、`run_command` は `sh` を利用。未導入だと当該ツールのみ失敗（ビルドには無関係）。
