# Functional Design — U5 CLI Frontend (consolidated)

> 設計判断（おまかせ＝推奨）。U5は統合unit: U1-U4を結線し、`agent.Frontend` と `guardrail.Confirmer` を実装。入力UIは **bubbletea フルTUI（REPL）**、単発実行はプレーン・ストリーミング出力（App決定 2026-06-08）。

## 担当ストーリー
US-1.1（CLI入力: 単発/REPL）, US-1.2（履歴・思考・アクション可視化）, US-1.3（Ctrl-C中断）, US-5.2（危険操作の確認UI）, US-6.1（接続失敗UX）, 全体結線。

## ドメイン/コンポーネント
- **App**（`internal/cli`）: Config→LLM Client(U2)→Tools Registry(U3)→Evaluator/ToolDispatcher(U3)→Agent Runner(U4) を組み立てる。
- **plainFrontend**: `agent.Frontend` 実装。イベントを `io.Writer`（stdout）へラベル付きで逐次出力（単発・非TTY向け）。
- **promptConfirmer**: `guardrail.Confirmer` 実装。TTYで reason を提示し y/N を読む。**非対話時は confirmer=nil → ガードレールがブロック**（フェイルクローズ, R8）。
- **tuiModel**（bubbletea）: REPL。textinput + viewport(履歴)。Enterで Runner をgoroutine実行、イベントは tea.Msg で受信し履歴へ追記（ストリーミング）。確認は confirming 状態でモーダル y/N。Ctrl-Cで ctx キャンセル（US-1.3）。
- **Run(ctx, args, stdout, stderr) int**: 引数に指示があれば単発、無ければREPL(TUI)。非TTYかつ指示無しはエラー。

## ビジネスルール（テスト可能な表明）
- **R1 モード選択**: `shiroutocode "<指示>"` → 単発（plain）。引数なし＋TTY → REPL(TUI)。引数なし＋非TTY → エラー終了（使い方提示）。
- **R2 単発出力**: plainFrontend は assistant text を逐次、ツール呼び出し/結果/ステップを区別可能に出力（US-1.2）。完了時に変更ファイル要約。
- **R3 確認**: 単発(非TTY)は confirmer=nil。単発(TTY)/REPL は実 confirmer。`promptConfirmer` は "y"/"yes"(大小無視)のみ true、他は false。
- **R4 接続エラー（US-6.1）**: Runner が Failed かつ LLMError(Unreachable/Timeout/ModelNotFound) のとき、原因+対処（起動/URL/モデル）を提示。内部詳細は出さない（SECURITY-09）。終了コード≠0。
- **R5 中断（US-1.3）**: TUIで Ctrl-C → ctx キャンセル → Runner が Aborted → 中断表示。
- **R6 TUI更新（テスト対象）**: model.Update は textMsg→履歴追記、stepMsg→進行更新、confirmReqMsg→confirming状態、y/n キー→replyチャネルへ送信し通常状態へ、doneMsg→結果表示。

## テスト観点（TDD）
| 観点 | 種別 |
|---|---|
| モード選択（単発/REPL/非TTYエラー） | unit |
| App 結線（registry にツール登録、dispatcher 構築） | unit |
| plainFrontend 出力フォーマット（区別・逐次） | unit |
| promptConfirmer y/N 解釈（injected reader） | unit |
| 単発 run: fake LLM で完了→stdoutに要約 | integration |
| 接続エラー時の終了コードとメッセージ | unit |
| tuiModel.Update: text/step/confirm/done 遷移 | unit |

## 技術スタック
- bubbletea + bubbles(textinput/viewport) + lipgloss（REPL TUI、利用者決定）。単発はプレーン（依存なし）。
- 中断: `context` + bubbletea KeyCtrlC。
- 実LM Studio E2Eは手動/Build and Test（本環境では未起動のため自動E2Eはbuild+smokeで代替）。

## 拡張コンプライアンス
SECURITY-09（接続エラー一般化）, SECURITY-11(ツールは U3 経由のみ＝Runner/dispatcher), SECURITY-15(非対話フェイルクローズ), SECURITY-03(マスキングLogger), PBT-09（promptConfirmer解釈をPBT可）。
