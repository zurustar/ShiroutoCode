# NFR Requirements + Design — U5 CLI Frontend (consolidated)

**Unit**: U5（統合）。規約: TDD + PBT。（おまかせ＝推奨）

## NFR
| 領域 | 判定/内容 |
|---|---|
| Scalability/Availability | N/A |
| Performance (P1) | ストリーミング即時表示（plain/TUIとも）。TUIは bubbletea の Msg/Cmd で再描画（生Println混在しない） |
| Reliability (R1) | 接続エラー/中断で安全に終了（フェイルクローズ）。終了コードで成否を表現 |
| Usability (U1) | 履歴/思考/アクションを区別表示、危険操作の明示確認、接続失敗の分かりやすい案内 |
| Maintainability (M1) | App結線/plainFrontend/confirmer は TUI非依存で単体テスト可能。TUIは Update をロジックテスト |
| Security | S-09 エラー一般化、S-11 ツールはU3経由、S-15 非対話フェイルクローズ |

## Tech Stack
| # | 決定 | 採用 |
|---|---|---|
| T1 | TUI | `charmbracelet/bubbletea` + `bubbles`(textinput/viewport) + `lipgloss`（REPL） |
| T2 | 単発/非TTY | プレーン `io.Writer` 出力（依存なし） |
| T3 | TTY判定 | `golang.org/x/term`（IsTerminal） |
| T4 | 中断 | `context` + KeyCtrlC / `os/signal`（単発時SIGINT） |
| T5 | テスト | `testing` + `rapid` + fake LLM/Dispatcher、Update関数の直接テスト |

## NFR Design パターン
- **P1 Bridge**: agent.Frontend(plain/tea) と guardrail.Confirmer(prompt/tea/nil) を mode別に注入。コアApp(BuildCore)はUI非依存。
- **P2 TUIイベント**: agent goroutine → `chan tea.Msg` → bubbletea `listen` Cmd で受信・再購読。confirmは replyチャネルで agent goroutine を一時ブロック。
- **P3 フェイルクローズ**: 非対話は confirmer=nil（U3がブロック）。エラーは一般化して終了コード≠0。
- **論理部品**: App, plainFrontend, promptConfirmer, teaFrontend, teaConfirmer, tuiModel, Run。

## テスト可能な受け入れ観点
- 単発で fake LLM 完了 → stdout に最終要約、終了コード0。
- 接続不可（fake LLM が Unreachable）→ 終了コード≠0、案内メッセージ、内部情報なし。
- 非対話で危険操作 → 実行されない（U3 + nil confirmer）。
- tuiModel.Update の状態遷移（text/step/confirm/done, Ctrl-C）。
- `go build ./...` で単一バイナリ生成、`shiroutocode`（指示なし・非TTY）は使い方を出して非0終了。
