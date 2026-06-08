# Units of Work — ShiroutoCode

**決定（Units Generation Q&A 2026-06-08）**: Q1=A(5 units 漸進) / Q2=A(U1→U5順) / Q3=A(各unitは単体・PBT greenで完了、E2EはU5完成時) / Q4=A(コード組織そのまま) / Q5=なし。

## デプロイモデル
- **単一バイナリ（モノリス）**: 独立デプロイ単位（Service）は **1つ**（`shiroutocode`）。
- 以下の「Unit of Work」は**開発・Construction ループの単位**であり、論理モジュール（Goパッケージ）にほぼ一致する。独立デプロイ単位ではない。

## コード組織戦略（Greenfield）
単一Goモジュール・単一バイナリ:
```
shiroutocode/                       # module: github.com/zurustar/shiroutocode
├── go.mod / go.sum
├── cmd/shiroutocode/main.go        # エントリ（→ internal/cli.Run）
└── internal/
    ├── config/     # U1
    ├── log/        # U1
    ├── llm/        # U2
    ├── tools/      # U3
    │   ├── file/  terminal/  git/  web/
    ├── guardrail/  # U3
    ├── agent/      # U4
    └── cli/        # U5
```
- 依存方向は上位→下位の一方向（Application Design レイヤリングに一致）。
- CLIフレームワークは標準 `flag`（暫定, SECURITY-10）。

---

## U1. Foundation
- **責務**: 構成管理（優先順位 flag>env>file>default、URL検証）と構造化ログ（機微情報マスキング）。全unitの土台。
- **コンポーネント**: C6 Config, C7 Logging
- **パッケージ**: `internal/config`, `internal/log`
- **主な成果**: `config.Load`, `log.Logger`
- **完了条件（Q3=A）**: 単体テスト + PBT（設定優先順位の決定、マスキング）green。
- **関連拡張**: SECURITY-05(URL検証), SECURITY-09(デフォルト認証情報なし), SECURITY-03(マスキング)

## U2. LLM Connectivity
- **責務**: LM Studio（OpenAI互換REST）連携、SSEストリーミング、function calling能力判定、接続エラー整形。
- **コンポーネント**: C3 LLM Client
- **パッケージ**: `internal/llm`
- **依存**: U1（Config, Logging）
- **主な成果**: `llm.Client`（`Complete`/`Capabilities`/`Stream`）
- **完了条件**: 単体テスト + PBT（ハイブリッド応答パーサ）green。SSE/エラーはモックHTTPで検証。
- **関連**: FR-2, NFR-3, US-2.1/2.2/6.1, SECURITY-09/13

## U3. Tools & Guardrail
- **責務**: ツール実装（File/Terminal/Git/Web）と、その単一実行窓口であるガードレール（スコープ限定・危険判定・フェイルクローズ）。
- **コンポーネント**: C4 Tool Layer, C5 Guardrail
- **パッケージ**: `internal/tools/...`, `internal/guardrail`
- **依存**: U1（Config: ワークスペースルート/ポリシー, Logging）
- **主な成果**: `tools.Tool`/`Registry`, `guardrail.Evaluator`/`ToolDispatcher`
- **完了条件**: 単体テスト + PBT（危険判定の性質、パス正規化、スコープ逸脱）green。
- **設計上の不変条件**: ツール実行は ToolDispatcher のみを入口とする（バイパス不可）。
- **関連**: FR-4/FR-5, US-4.1〜4.5, US-5.1〜5.3, US-6.2, SECURITY-05/11/13/15

## U4. Agent Engine
- **責務**: plan→act→observe ループ、会話履歴（メモリ）、ステップ管理・終了条件、ツールはU3経由で実行、進行/結果をFrontend Portへ。
- **コンポーネント**: C2 Agent Engine
- **パッケージ**: `internal/agent`
- **依存**: U2（LLM）, U3（ToolDispatcher）, U1（Logging）, Frontend Port（U5が実装、interfaceとして本unitで定義）
- **主な成果**: `agent.Runner`, `Frontend` Port インタフェース定義
- **完了条件**: 単体テスト + PBT（終了条件: 完了/最大ステップ/中断の不変条件）green。LLM/Dispatcher/Frontend はモック。
- **関連**: FR-3, US-3.1〜3.3, NFR-4

## U5. CLI Frontend（統合）
- **責務**: 対話REPL + 単発実行、イベントの逐次レンダリング、Ctrl-C中断、危険操作の確認プロンプト（非TTYは安全停止）、接続エラーUX。Frontend Port を実装し全体を結線（main）。
- **入力インタフェース決定（2026-06-08, 利用者選択=C）**: **フルTUI = `charmbracelet/bubbletea`**（+ `bubbles` textinput/textarea で行・複数行編集と履歴、`lipgloss` で装飾）。
  - **含意（U5 NFR/設計で必ず扱う）**: bubbletea は Elm系の更新ループ + 代替スクリーン。エージェントのストリーミング出力は **bubbletea の Msg/Cmd 経由でモデルへ流し込んで再描画**する設計にする（生の `fmt.Println` 直書きと混在させない）。
  - **モード差**: 対話REPLは bubbletea プログラム。**単発実行（`shiroutocode "指示"`）は TUI を使わずプレーンなストリーミング標準出力**（パイプ/非TTY/CI 向け、US-5.2の非対話=安全停止と整合）。
  - 依存増は SECURITY-10 上のトレードオフを許容（利用者判断）。確認プロンプトは bubbles のリスト/確認UIで実装。
- **コンポーネント**: C1 CLI
- **パッケージ**: `internal/cli`, `cmd/shiroutocode`
- **依存**: U1〜U4 すべて（統合点）
- **主な成果**: `cli.Run`, REPL/単発、`main`
- **完了条件**: 単体テスト green ＋ **この時点で end-to-end 動作確認**（LM Studio接続〜マルチファイル編集の主要シナリオ）。
- **関連**: FR-1/FR-6, US-1.1〜1.3, US-3.2(表示), US-5.2(確認UI), US-6.1

---

## 実装順序（Q2=A）
**U1 → U2 → U3 → U4 → U5**（各 unit を design+code まで完了させてから次へ。Construction はこの順でループ）
