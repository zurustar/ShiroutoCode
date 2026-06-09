# Domain Entities — U3 Tools & Guardrail

> 技術非依存のドメインモデル。Go型名は参考。

## ツール層（C4）

### E1. Tool（共通契約）
| 要素 | 説明 |
|---|---|
| `Name()` | ツール名（例 `read_file`/`write_file`/`run_command`/`git`/`web_fetch`） |
| `Spec()` | LLMへ提示する ToolSpec（名前/説明/引数JSON Schema）。U2の `llm.ToolSpec` と整合 |
| `Execute(ctx, args) (ToolResult, error)` | 実行。argsは正規化済みmap |

### E2. ToolResult
| フィールド | 型 | 説明 |
|---|---|---|
| `Output` | 文字列 | 人間/LLM向け結果要約 |
| `ExitCode` | int | Terminal用（0=成功） |
| `Stream` | <-chan StreamLine | 任意: 逐次出力（Terminal, App Q4=B） |
| `Changed` | []string | 変更したパス（File/Git） |

### E3. 各ツールの引数（概略）
- `read_file`: `{path}`
- `write_file`: `{path, mode: "create"|"overwrite"|"edit"|"delete", content?, old_string?, new_string?}`（Q1=A）
- `run_command`: `{command, args[]?}` または `{command_line}`（シェル経由可否はR）
- `git`: `{op, args[]}`（op: status/diff/log/add/commit/branch/checkout/push/...）
- `web_fetch`: `{url}`（GET, Q6=A）

### E4. ActionKind（ガードレール評価の対象種別）
`FileRead` / `FileWrite` / `FileDelete` / `Command` / `GitOp` / `WebFetch`。

### E5. Action（評価入力）
| フィールド | 型 | 説明 |
|---|---|---|
| `Kind` | ActionKind | 種別 |
| `Tool` | 文字列 | ツール名 |
| `Paths` | []string | 関与パス（File/Git） |
| `CommandLine` | 文字列 | Command/Git の実行内容 |
| `URL` | 文字列 | WebFetch |

## ガードレール層（C5）

### E6. Decision（判定, Q2=A）
`Allow` / `Confirm` / `Deny` の3値。`Confirm`/`Deny` は **理由（人間可読）** を伴う。

### E7. GuardrailPolicy（U1 Config由来 + 既定ルール）
| フィールド | 説明 |
|---|---|
| `WorkspaceRoot` | スコープの基準（絶対・正規化） |
| `ConfirmMode` | `prompt`(確認) / `deny`(即拒否) … Confirm時の挙動 |
| `ExtraDenyPatterns` | 利用者追加のdenylist |
| `NonInteractivePolicy` | `stop`/`deny`（非対話時の安全側挙動, Q7=A） |

### E8. Rule（判定規則）
| フィールド | 説明 |
|---|---|
| `Match` | Action にマッチする述語（種別/パス/コマンドパターン/URL） |
| `Decision` | マッチ時の Allow/Confirm/Deny |
| `Reason` | 理由テキスト |
`Evaluator` は順序付き Rule 群 + スコープ検査 + フェイルクローズ既定で構成。

### E9. Confirmer（確認の境界, Q7=A）
`Confirm(ctx, action Action, reason string) (bool, error)`。U5（CLI/TUI）が実装。**非対話時は Deny 相当**（フェイルクローズ）。

### E10. ToolDispatcher（単一インターセプタ, App Q5=A）
`Dispatch(ctx, call ToolCall) (ToolResult, error)`。内部: Action化 → Evaluate → (Confirm時)Confirmer → Allowなら Registry経由でTool実行。**唯一のツール実行入口**。

### E11. Registry
`Register(Tool)` / `Get(name)` / `Specs()`（LLMへ渡すスキーマ列、U2連携）。

## 関係
```text
Agent(U4) → ToolDispatcher(E10) → Evaluator(E8/E6) →(Confirm) Confirmer(E9)
                                 →(Allow) Registry(E11) → Tool(E1) → ToolResult(E2)
Policy(E7) は Evaluator と スコープ検査に供給（WorkspaceRoot 等は U1 Config）
```
