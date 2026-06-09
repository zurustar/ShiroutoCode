# Code Summary — U5 CLI Frontend

**生成日**: 2026-06-10 / TDD。**全テスト green / -race クリーン**。単一バイナリ生成成功（~10MB）。

## 生成/変更ファイル
| パス | 種別 | 内容 |
|---|---|---|
| `cmd/shiroutocode/main.go` | 新規 | エントリ: TTY判定, signal→ctx, env収集 → cli.Run |
| `internal/cli/app.go` | 新規 | `BuildCore`（config→llm→tools registry→guardrail→`newRunner`） |
| `internal/cli/plain.go` | 新規 | `plainFrontend`（agent.Frontend, 単発/非TTY出力） |
| `internal/cli/confirm.go` | 新規 | `promptConfirmer`（guardrail.Confirmer, y/N）, `parseYes` |
| `internal/cli/run.go` | 新規 | `Run`（モード選択）, `extractPrompt`, `runSingleShot`, `reportResult`, `failureMessage`（US-6.1） |
| `internal/cli/tui.go` | 新規 | bubbletea `tuiModel`, `teaFrontend`/`teaConfirmer`, `runREPL` |
| `internal/agent/agent.go` | 変更 | `Result.Err` 追加（接続エラー原因の伝達） |

## ストーリー対応
| Story | 実装 |
|---|---|
| US-1.1 CLI入力（単発/REPL） | run.go モード選択, tui.go textinput |
| US-1.2 履歴/思考/アクション可視化 | plainFrontend / tuiModel（区別表示・逐次） |
| US-1.3 中断 | main signal→ctx, tui Ctrl+C→cancel |
| US-5.2 確認UI | promptConfirmer / teaConfirmer（confirming状態） |
| US-6.1 接続失敗UX | failureMessage（LLMError.UserMessage, 内部非露出） |
| 全体結線 | BuildCore + newRunner |

## 検証（E2E相当）
- `go build -o shiroutocode ./cmd/shiroutocode` 成功（単一バイナリ）。
- smoke: model未設定 → 設定エラー exit=2。
- smoke: model設定・LM Studio未起動 → リトライ後「LM Studio に接続できません…」exit≠0（US-6.1 を実バイナリで確認）。
- 実LM Studio接続の完全E2E（マルチファイル編集）は LM Studio 起動環境で手動確認（Build and Test に記載）。

## 拡張コンプライアンス
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-09 | ✔ | 接続エラーは一般化文言、内部詳細非露出（テスト済） |
| SECURITY-11 | ✔ | ツールは BuildCore→ToolDispatcher 経由のみ |
| SECURITY-15 | ✔ | 非対話 confirmer=nil でブロック（フェイルクローズ） |
| SECURITY-10 | 部分 | TUI依存(bubbletea等)を追加（利用者決定）。単発系統は標準のみ |
| PBT-09 | ✔ | parseYes をPBT |

## 既知の polish（将来）
- 単発時の操作ログ(WARN/ERROR)が stderr に出る（既定 log-level=info）。クリーンな既定にしたい場合は log-level=error 既定化を検討。
- bubbletea TUI の完全な対話E2Eは自動テスト対象外（Update遷移は単体テスト済）。
