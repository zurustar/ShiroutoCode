# Code Summary — U3 Tools & Guardrail

**生成日**: 2026-06-10 / **規約**: TDD + PBT(rapid)。**全テスト green / -race クリーン**。新規Go依存なし（`git` CLI 実行）。

## 生成ファイル
### `internal/tools/`
| パス | 内容 |
|---|---|
| `tool.go` | `Tool`/`Registry`/`ToolResult`/`ToolCall` |
| `file.go` | `FileTool`(create/overwrite/edit/delete, 原子的書込P4), `ReadFileTool` |
| `terminal.go` | `TerminalTool`(`os/exec`, setpgid, Cancel=group kill, timeout, capWriter出力上限) |
| `git.go` | `GitTool`(`git` CLI) |
| `web.go` | `WebTool`(`net/http` GET, http(s)のみ, サイズ上限, リダイレクト制限) |
| `tools_test.go`/`web_test.go` | 単体 |

### `internal/guardrail/`
| パス | 内容 |
|---|---|
| `types.go` | Action/Decision/Policy/Confirmer/BlockedError |
| `scope.go` | `resolveWithin`/`evalExisting`(P1 symlink解決) |
| `rules.go` | denylist/confirmlist マッチャ + `defaultRules` テーブル(P2) |
| `evaluator.go` | `Evaluator.Evaluate`(R3-R10, スコープ+ルール+フェイルクローズ) |
| `dispatcher.go` | `ToolDispatcher`(P5/R1/R2/R8, 単一窓口) + `toAction` |
| `*_test.go` | 単体 + PBT |

## 設計対応（主）
| ルール/パターン | 実装 | テスト |
|---|---|---|
| R1 単一窓口 | ToolDispatcher.Dispatch のみ実行 | dispatcher_test |
| R2/R8 実行条件 | Allow / Confirm+yes のみ | TestDispatchExecutionInvariantPBT |
| R3 スコープ封じ込め | resolveWithin(EvalSymlinks) | TestScopeContainmentPBT/Symlink |
| R4 denylist | rmRootish/forkBomb/deviceWrite/mkfs/systemPower/pipeToShell | TestCommandDenylistPBT |
| R5 confirmlist | sudo/recursivePerm | TestCommandConfirmAndAllow |
| R6 git危険操作 | gitDestructive/gitPush | TestGitDestructivePBT |
| R7 web制限 | scheme/GET/サイズ上限 | TestWebScheme/SizeCap |
| R9 フェイルクローズ | 未知/解決不能→Confirm、confirmerエラー→Block | evaluator/dispatcher tests |
| P3 プロセスgroup kill | exec Cancel + setpgid | TestTerminalTimeoutKills |
| P4 原子的書込 | writeAtomic(temp→rename) | TestFileCreateReadEditDelete |

## 拡張コンプライアンス（U3 Code）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 | ✔ | 引数/パス/コマンド/URL検証 |
| SECURITY-11 | ✔(中核) | guardrail 専用パッケージ・単一インターセプタ |
| SECURITY-13 | ✔ | LLM由来引数を評価後のみ実行 |
| SECURITY-15 | ✔ | フェイルクローズ・原子的書込・group kill |
| PBT-09 (rapid) | ✔ | スコープ/denylist/git/dispatch をPBT |
| SECURITY-10 | ✔ | 新規Go依存ゼロ（os/exec, net/http, path/filepath） |
| SECURITY-03 | 継承(U1) | 実行ログはマスキングLogger経由 |

## 注記
- `git` ツールは外部 `git` 実行ファイルに依存（不在時はそのツールのみ機能低下）。
- ターミナルの setpgid/group kill は Unix前提（darwin/linux）。Windows対応は将来課題。
- 実LM Studio/実コマンドのE2EはU5/Build and Test。
