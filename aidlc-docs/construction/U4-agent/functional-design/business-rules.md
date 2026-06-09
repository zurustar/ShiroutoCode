# Business Rules — U4 Agent Engine

> テスト可能な表明（TDD）。`[PBT]` は rapid 候補。担当: US-3.1/3.2/3.3, NFR-4。

## ループ（plan→act→observe, FR-3）

### R1. ステップ反復
- 表明: 各ステップで (1) 現在の会話＋ツールspecで LLM.Complete → (2) 応答を集約（テキスト+ツール呼び出し）→ (3) ツール呼び出しがあれば Dispatcher 経由で実行し観測を会話へ追記 → 次ステップ。
- 表明: LLMテキストは `Frontend.OnAssistantText` で逐次転送（US-2.2）。

### R2. 完了判定（US-3.1）
- 表明: 応答に**ツール呼び出しが無い**とき、その最終テキストを `Summary` として `Completed` で終了。
- 表明: 完了時に変更ファイル一覧（実行中の ToolResult.Changed 集約）を Result に含める。

### R3. 最大ステップ（US-3.3, 暴走防止）
- 表明: ステップ数が `maxSteps`（Config, FR-6）に達しても未完なら `StoppedMaxSteps` で停止（未完である旨）。
- `[PBT]` 常にツール呼び出しを返すLLMでは、実行ステップ数 == maxSteps で停止する（無限ループしない）。

### R4. 中断（US-1.3, NFR-3）
- 表明: `ctx` がキャンセルされたら、進行中のLLM/ツールを中断し `Aborted`（または ctx.Err()）で速やかに終了。
- 表明: ストリーム/ツール実行のエラーで危険側に進まない（NFR-4）。

## ツール実行と観測

### R5. ツールは Dispatcher 経由のみ（U3連携, R1単一窓口）
- 表明: Runner は `tools.Tool.Execute` を直接呼ばず、必ず `Dispatcher.Dispatch` を使う。
- 表明: 各ツール呼び出しは `Frontend.OnToolCall`、結果は `OnToolResult` に通知。

### R6. 観測の会話への反映（モード別）
- function モード: assistant メッセージ（ToolCalls付き）＋ 各ツール結果を `role=tool`(ToolCallID付き) で追記。
- json モード: assistant メッセージ（JSON本文）＋ ツール結果を `role=user`（"Tool <name> result:\n<output>"）で追記。
- 表明: ツールが**ブロック**（BlockedError）された場合も、その旨を観測として追記しループ継続（モデルが代替を選べる）。ステップは消費する。

### R7. システムプロンプト
- 表明: 既定のシステムプロンプト（plan→act→observe方針、ツール使用、最終回答で終了）を会話先頭に置く。空のユーザー指示は実行しない。

## 横断
- 会話履歴はメモリのみ（永続化なし, App Q6=A）。
- LLM/ツール/フロントは DI（テスト容易, NFR-6）。ログはU1マスキング前提。
