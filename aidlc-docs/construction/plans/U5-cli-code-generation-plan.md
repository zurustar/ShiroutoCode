# Code Generation Plan — U5 CLI Frontend（CONSTRUCTION）

**Unit**: U5（`internal/cli` + `cmd/shiroutocode`）。統合unit。TDD。
**依存**: U1-U4 全て。追加依存: bubbletea/bubbles/lipgloss(+x/term)（利用者決定: TUI）。

## ステップ（実施済み）
- [x] Step 1: agent.Result に Err 追加（接続エラー表示用, US-6.1）
- [x] Step 2: `cli/app.go` BuildCore（config→llm→tools→guardrail→runner 結線）
- [x] Step 3: `cli/plain.go` plainFrontend（agent.Frontend, 単発出力）
- [x] Step 4: `cli/confirm.go` promptConfirmer（guardrail.Confirmer, y/N）
- [x] Step 5: `cli/run.go` Run（モード選択 / 単発 / 終了コード / 接続エラー文言）
- [x] Step 6: `cli/tui.go` bubbletea model + teaFrontend/teaConfirmer + runREPL
- [x] Step 7: `cmd/shiroutocode/main.go`（TTY判定, signalでctx, env）
- [x] Step 8: テスト（confirmer/plain/extractPrompt/run modes/single-shot integration/TUI Update/接続エラー）
- [x] Step 9: ローカル検証（build バイナリ + smoke + test -race + gofmt/vet）

## 完了条件
`go test ./... -race` green、`go build -o shiroutocode ./cmd/shiroutocode` 成功、smoke（設定不足/接続失敗）で適切な終了コードと文言。
