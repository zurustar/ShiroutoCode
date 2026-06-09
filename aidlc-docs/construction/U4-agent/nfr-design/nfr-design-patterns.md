# NFR Design Patterns — U4 Agent Engine

> 技術: Go標準 + U1/U2/U3。TDDで先行テスト。論理インフラ部品なし。

## P1. 停止性（NFR-R1）
- ループは `for step:=1; step<=maxSteps; step++`。各反復頭で `ctx.Err()` を確認。完了（ツールなし）/上限/キャンセルの3経路のみで抜ける。
- **テスト(PBT)**: 常時ツール要求の fake LLM で、Dispatch呼び出し回数 ≤ maxSteps、戻りは StoppedMaxSteps。

## P2. キャンセル伝播（NFR-R2/US-1.3）
- 親 `ctx` を `llm.Complete` と `dispatcher.Dispatch` に渡す。両者が ctx 連動で中断。ループ頭でも確認し `Aborted`。

## P3. ストリーミング転送（NFR-P1）
- `llm.CollectStreaming(stream, fe.OnAssistantText)` で受信即転送＋集約。バッファ無し。

## P4. フェイルクローズ（NFR-R2）
- Complete失敗→Failed（ctx由来はAborted）。ツール失敗/Blocked→観測化して継続（危険側に進まない）。stream は defer Close。

## P5. 依存注入（NFR-M1）
- `Runner` は `llm.LLMClient` / `Dispatcher`(interface) / `Frontend`(interface) / `Registry` / `Logger` を保持。テストは fake を注入。

## 適用しないパターン
- リトライ（U2が担当）、キャッシュ、並行実行（MVPは逐次1セッション）= N/A。

## 論理コンポーネント
- **LC1 Runner**: ループ統括（P1/P2/P3/P4）。
- **LC2 Frontend(port)** + **noopFrontend**: 進行通知（P5/NFR-U1）。
- **LC3 specBuilder**: registry→[]llm.ToolSpec。
- **LC4 conversation**: メッセージ追記（観測のモード別整形, Functional R6）。
