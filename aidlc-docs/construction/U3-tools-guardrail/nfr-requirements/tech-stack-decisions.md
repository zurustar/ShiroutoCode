# Tech Stack Decisions — U3 Tools & Guardrail

| # | 決定事項 | 採用 | 理由 |
|---|---|---|---|
| T1 | ターミナル実行 | **標準 `os/exec`**、既定は引数配列、必要時 `sh -c` | 依存なし。denylist はコマンド文字列全体（shell連結含む）に適用 |
| T2 | 実行制御 | `context` 連動タイムアウト + プロセス終了 + 出力上限 | NFR-R2。既定タイムアウト 120s（設定可） |
| T3 | Git | **`git` CLI を `os/exec`** | 依存なし。ユーザー環境のgit/認証を利用 |
| T4 | Web | **標準 `net/http`**（GET、サイズ上限、リダイレクト制限） | 依存なし。SSRF/巨大応答に上限で対処 |
| T5 | パス処理 | 標準 `path/filepath` + `filepath.EvalSymlinks` | スコープ封じ込め（R3） |
| T6 | denylist | データ駆動ルール表 + Config `ExtraDenyPatterns` | 保守性・拡張性 |
| T7 | テスト | `testing` + `rapid` + `t.TempDir()` + `httptest` | TDD。ファイル/コマンド/HTTP をhermeticに |

## 依存方針
- **U3で増える本体依存は無し**（標準ライブラリ + 外部 `git` 実行ファイル）。`git` 不在環境では git ツールのみ機能低下（エラーを明示）、他ツールは動作。
- GPL-3.0互換を維持（新規Go依存なし）。

## 既定値（暫定, 設定で上書き可）
- コマンドタイムアウト: 120s
- コマンド/Web 出力サイズ上限: 例 1 MiB（取得は打ち切り）
- HTTPリダイレクト最大: 例 5

## 設計メモ（Code Generation向け）
- `internal/tools`（file/terminal/git/web）と `internal/guardrail`（Evaluator/ルール表/ToolDispatcher）に分離（SECURITY-11）。
- Evaluator・スコープ・denylist は純粋関数中心 → PBT。Tool.Execute は OS副作用 → TempDir/echo 等でテスト。
- `Confirmer` フェイク（yes/no/非対話）でディスパッチ分岐をテスト。
