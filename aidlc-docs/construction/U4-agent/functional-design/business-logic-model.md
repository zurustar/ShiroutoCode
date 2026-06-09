# Business Logic Model — U4 Agent Engine

> 技術非依存フロー。TDD前提。

## F1. Run フロー
```text
Run(ctx, task):
  if task.Prompt 空 → エラー（R7）
  msgs = [system(systemPrompt), user(task.Prompt)]
  changed = []
  for step := 1..maxSteps:
    if ctx.Err() → return Aborted
    req = {Messages: msgs, Tools: registrySpecs(), ToolMode, Stream:true}
    stream, err = llm.Complete(ctx, req); if err → Failed(err)
    res, err = CollectStreaming(stream, fe.OnAssistantText); stream.Close()
        if err → (ctx由来ならAborted) else Failed
    fe.OnStep(step, maxSteps)
    if len(res.ToolCalls)==0:
        return Completed{Summary:res.Text, ChangedFiles:changed, Steps:step}
    # act + observe
    appendAssistant(msgs, res, stream.Mode())
    for tc in res.ToolCalls:
        fe.OnToolCall(tc.Name, tc.Args)
        out, derr = dispatcher.Dispatch(ctx, ToolCall{tc.Name, tc.Args})
        fe.OnToolResult(tc.Name, out.Output, derr)
        changed += out.Changed
        appendObservation(msgs, tc, out, derr, stream.Mode())   # BlockedErrorも観測化(R6)
  return StoppedMaxSteps{ChangedFiles:changed, Steps:maxSteps}
```

## F2. 観測追記（R6）
```text
function: assistant{content:res.Text, tool_calls:res.ToolCalls}; 各 tool{tool_call_id, content:out|err}
json:     assistant{content:res.Text}; 各 user{"Tool <name> result:\n"+(out|err)}
Blockederr: content = "操作はブロックされました: "+reason
```

## F3. ツールspec生成
```text
registrySpecs(): registry の各 Tool → llm.ToolSpec{Name, Description, Parameters: 緩いobjectスキーマ}
（厳密スキーマは将来。引数検証は実行時にツール/ガードレールが行う）
```

## テスト観点（TDD）
| 観点 | 種別 | ルール |
|---|---|---|
| 単発完了（ツールなし→Completed, OnAssistantText） | unit | R2 |
| ツール→完了（2ステップ, changed集約, Dispatch経由） | unit | R1/R5 |
| 常時ツール→maxStepsで停止 | PBT | R3 |
| ctxキャンセル→Aborted（速やか） | unit | R4 |
| ブロック観測でループ継続 | unit | R6 |
| 空プロンプト→エラー | unit | R7 |
| OnStep/OnToolCall/OnToolResult 通知 | unit | R1/R5 |

## 実装メモ（Code Generation向け）
- パッケージ `internal/agent`。`Runner` に依存をDI。`Frontend` Port と no-op 実装を定義。
- `llm.CollectStreaming(stream, onText)` を利用（U2に追加：既存 `Collect` は `CollectStreaming(s,nil)` へ委譲）。
- テスト: fake `llm.LLMClient`（スクリプト化ストリーム）＋ fake Dispatcher。実LLM/実ツール不要。

## 拡張コンプライアンス（U4 Functional Design）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-11 | 反映 | ツールは必ず Dispatcher(U3) 経由（R5） |
| SECURITY-15 | 反映 | フェイルクローズ（エラー時危険側に進まない）、max steps |
| PBT-09 | 反映 | R3（停止性）をPBT |
| SECURITY-03 | 継承(U1) | ログマスキング |
| その他 | N/A | ネット/DB/認証なし |
