# Domain Entities — U4 Agent Engine

> 設計判断（おまかせ＝推奨）: 完了判定=ツール呼び出しが無くなった時点 / 観測はメッセージ追記 / フロントは Frontend Port で疎結合 / 中断は ctx。

## E1. Task
| フィールド | 説明 |
|---|---|
| `Prompt` | ユーザー指示（自然言語） |
| （workspace等は Config 経由でツール側が保持） |

## E2. Status
`Completed`（最終回答到達）/ `StoppedMaxSteps`（上限到達・未完）/ `Aborted`（ctxキャンセル）/ `Failed`（回復不能エラー）。

## E3. Result
| フィールド | 説明 |
|---|---|
| `Status` | E2 |
| `Summary` | 最終アシスタントテキスト（変更要約） |
| `ChangedFiles` | 実行中に変更されたパス（ツール結果から集約） |
| `Steps` | 実行ステップ数 |

## E4. Frontend（Port, U4が定義 / U5が実装）
エージェントの進行を表示するための疎結合インタフェース（ガードレールの Confirmer とは別）。
| メソッド | 用途 |
|---|---|
| `OnAssistantText(delta)` | LLMテキストの逐次表示（US-2.2/1.2） |
| `OnToolCall(name, args)` | ツール呼び出し開始（US-1.2/4.x） |
| `OnToolResult(name, output, err)` | ツール結果/失敗 |
| `OnStep(current, max)` | ループ進行（US-3.2） |
- 既定は no-op 実装（テスト/非UI実行）。

## E5. Dispatcher（依存・U3）
`Dispatch(ctx, tools.ToolCall) (tools.ToolResult, error)`。guardrail.ToolDispatcher を注入。

## E6. Runner
`Run(ctx, Task) (Result, error)`。会話履歴（`[]llm.Message`、**メモリのみ** U6=A）を保持し plan→act→observe を回す。
- 依存注入: `llm.LLMClient`(U2), `Dispatcher`(U3), `*tools.Registry`(specs提示), `Frontend`(U5), `log.Logger`, maxSteps/toolMode/systemPrompt（U1 Config由来）。
