# Services & Orchestration — ShiroutoCode (Application Design)

> 「サービス」= プロセス内のオーケストレーション責務の単位（Goでは構造体/インタフェース）。ネットワークサービスではない（完全ローカル, NFR-2）。

## S1. Session / Runner（Application層の中心オーケストレータ）
- **担当コンポーネント**: Agent Engine (C2)
- **責務**: 1タスクの plan→act→observe ループを統括。会話履歴（メモリ）、ステップ管理、終了条件。
- **オーケストレーション**:
  1. `LLMService` に会話+ツールスキーマを渡し応答取得（ストリームは Frontend へ中継）
  2. 応答に ToolCall があれば `ToolDispatcher` に委譲（＝必ずGuardrailを通る）
  3. 実行結果（観測）を会話へ追記し次ステップへ。完了/上限/中断で終了
  4. 最終 `Result` と要約を Frontend へ

## S2. ToolDispatcher（Domain層・安全実行の単一窓口）
- **担当コンポーネント**: Guardrail (C5) ＋ Tool Layer (C4)
- **責務**: **すべてのツール実行の唯一の入口**（Q5=A）。バイパス不可。
- **オーケストレーション**:
  1. `Evaluator.Evaluate(action)` → `Allow / Confirm / Deny`
  2. `Confirm` → `Frontend.Confirm(...)`（非対話なら安全停止）／`Deny` → 実行せず理由を返す
  3. `Allow`（または確認OK）→ `Registry.Get(name).Execute(...)`
  4. 例外/判定不能は**フェイルクローズ**（危険側に進まない, SECURITY-15）
- **不変条件**: Agent Engine は C4 を直接呼ばず、必ず S2 を経由する（設計上・レビューで担保）。

## S3. LLMService（Infrastructure境界のアダプタ）
- **担当コンポーネント**: LLM Client (C3)
- **責務**: 会話メッセージ組み立て、ツールスキーマ提示、**ハイブリッドなツール呼び出し解釈（Q2=C）**、SSEストリーム→アプリ内イベントへの変換、接続エラーの整形（US-6.1）。
- **モード選択**: `Capabilities()` で function calling 可否を判定し、可ならネイティブ `tools`、不可ならプロンプト＋JSON/ReActパースへフォールバック。呼び出し側（S1）はモードを意識しない。

## S4. ConfirmationService（横断・フロント橋渡し）
- **担当**: Guardrail (C5) → Frontend Port (C1)
- **責務**: Guardrail の `Confirm` 判定をフロントの明示確認に橋渡し。CLIでは対話プロンプト、将来のVSCodeフロントではIPC経由のダイアログに置換可能（同一Port）。

## オーケストレーション・フロー（1ステップ）
```text
User → CLI(C1) → Runner(S1)
  S1 → LLMService(S3) → LM Studio (SSE) → S1（テキスト/ToolCall, ストリームはC1へ中継）
  S1 →(ToolCallあり) ToolDispatcher(S2)
        S2 → Evaluator(C5): Allow/Confirm/Deny
              Confirm → ConfirmationService(S4) → Frontend(C1) → ユーザー諾否
        S2 →(許可) Tool(C4).Execute → ToolResult（Terminalはストリーム）
  S1 ← 観測を会話へ追記 → 次ステップ or 終了 → Result を C1 へ
```

## サービス境界とテスト容易性（NFR-6 / PBT=rapid）
- S1/S2/S3 は依存をインタフェースで注入（モック可能）。
- **PBT対象**: Evaluator（危険判定の性質: 「ワークスペース外は常にAllowにならない」等）、パス正規化、ハイブリッド応答パーサ、設定優先順位の決定。
