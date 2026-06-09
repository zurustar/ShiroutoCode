# Code Generation Plan — U3 Tools & Guardrail（CONSTRUCTION / Part 1）

**Unit**: U3（`internal/guardrail` + `internal/tools`）。**最大・安全性の中核**。
**規約**: TDD（test-first）+ PBT(rapid)。本plan が単一の真実。
**依存**: U1（config/log）。U2非依存（ToolSpecの形だけ整合）。新規外部依存なし（`git` CLI を実行）。

## 担当ストーリー
US-4.1〜4.5（File/Terminal/Git/Web）, US-5.1〜5.3（自動承認/危険ブロック/スコープ）, US-6.2（フェイルクローズ）

## コード配置（確定）
```
internal/guardrail/
├── types.go        # Action, ActionKind, Decision, GuardrailPolicy, Confirmer, Rule
├── scope.go        # resolveWithin (P1)
├── rules.go        # defaultRules テーブル + マッチャ (P2)
├── evaluator.go    # Evaluator.Evaluate (R3-R10)
├── dispatcher.go   # ToolDispatcher (P5/R1/R2/R8)
internal/tools/
├── tool.go         # Tool interface, Registry, ToolResult, ToolCall
├── file.go         # FileTool (read/write/edit/delete, atomic P4)
├── terminal.go     # TerminalTool (os/exec, pgid kill, output cap, stream)
├── git.go          # GitTool (git CLI)
├── web.go          # WebTool (net/http GET, size/redirect limits)
└── *_test.go       # 各々に単体 + PBT
```
※ U3単体では `main` 無し。完了条件＝`go test ./...` green。

## 生成ステップ（TDD・順次）

### [ ] Step 1: tools 共通型（tool.go）
- `Tool`/`Registry`/`ToolResult`/`ToolCall`。コンパイル土台。

### [ ] Step 2: guardrail 型 + スコープ — テスト先行→実装（types.go, scope.go）
- RED: `resolveWithin` PBT（内rel=内, `../`=外, symlink脱出=外, 解決不能=非許可）。
- GREEN: 実装（P1）。

### [ ] Step 3: ルール表 + Evaluator — テスト先行→実装（rules.go, evaluator.go）
- RED: PBT（denylist変種→Deny, git force/hard→非Allow, web非GET/非http(s)→Deny）; unit（confirmlist→Confirm, 未知/不能→Confirm, 通常→Allow, スコープ連動）。
- GREEN: `defaultRules`/`Evaluator`（R3-R10）。

### [ ] Step 4: ディスパッチャ — テスト先行→実装（dispatcher.go）
- RED: PBT/unit（Allow→実行, Confirm+yes→実行, Confirm+no→未実行, Deny→未実行, 非対話→未実行）。フェイクTool/Confirmer。
- GREEN: `ToolDispatcher`（P5/R1/R2/R8）。

### [ ] Step 5: FileTool — テスト先行→実装（file.go）
- RED: read/create/overwrite/edit(一意置換)/delete; 原子的書込; edit一意でない→エラー（TempDir）。
- GREEN: 実装（P4/F3）。

### [ ] Step 6: TerminalTool — テスト先行→実装（terminal.go）
- RED: echo実行→stdout/exit; タイムアウト/ctxキャンセルで停止; 出力上限truncate。
- GREEN: `os/exec`+pgid kill（P3）。

### [ ] Step 7: GitTool + WebTool — テスト先行→実装（git.go, web.go）
- RED: git status（TempDirでinit）; web GET（httptest）, 非http(s)拒否, サイズ上限。
- GREEN: 実装（F5）。

### [ ] Step 8: コード要約ドキュメント
- `aidlc-docs/construction/U3-tools-guardrail/code/code-summary.md` / `test-summary.md`

### [ ] Step 9: ローカル検証
- `go build ./...` + `go test ./... -race`（rapid含む）green、`gofmt`/`go vet` クリーン。

### API/Frontend/DB/デプロイ
- **N/A**

## ストーリートレーサビリティ
| Story | ステップ | 完了条件 |
|---|---|---|
| US-4.1/4.2 | 5 | File read/write/edit/delete green |
| US-4.3 | 6 | Terminal 実行/中断/上限 green |
| US-4.4 | 7 | Git 実行 green |
| US-4.5 | 7 | Web GET 制限 green |
| US-5.1/5.2/5.3 | 2,3,4 | スコープ/denylist/dispatch green |
| US-6.2 | 3,4 | フェイルクローズ green |

## スコープ概算
- 9ステップ、`internal/guardrail`(5) + `internal/tools`(5) 実装 + 各テスト。新規外部依存なし（`git` 実行に依存）。
