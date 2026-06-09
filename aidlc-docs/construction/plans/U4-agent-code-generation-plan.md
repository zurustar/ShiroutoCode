# Code Generation Plan — U4 Agent Engine（CONSTRUCTION）

**Unit**: U4（`internal/agent`）。規約: TDD + PBT(rapid)。**新規外部依存なし**。
**依存**: U1(log/config), U2(llm), U3(tools/guardrail)。
**担当**: US-3.1/3.2/3.3（+ 表示は Frontend port 経由で U5 が実装）。

## コード配置
```
internal/llm/sse.go    # 追加: CollectStreaming(stream, onText)（既存 Collect を委譲に refactor）
internal/agent/
├── agent.go           # Frontend port, noopFrontend, Dispatcher iface, Runner, Result/Status, Run
├── conversation.go    # 観測のモード別メッセージ整形 + specBuilder
└── agent_test.go      # 単体 + PBT（fake LLM/Dispatcher/Frontend）
```
※ U4単体では `main` 無し。完了条件＝`go test ./... -race` green。

## 生成ステップ（TDD・順次）
### [x] Step 1: llm.CollectStreaming 追加（refactor）+ テスト
- `CollectStreaming(s, onText)`、`Collect=CollectStreaming(s,nil)`。既存テスト維持 + onText 呼び出しテスト。
### [x] Step 2: agent 型 + Frontend/Dispatcher/Runner 骨組み（agent.go）
### [x] Step 3: ループ — テスト先行→実装
- RED: 単発完了 / ツール→完了 / maxSteps(PBT) / ctxキャンセル / ブロック継続 / 空プロンプト / 通知。fake LLM(スクリプトstream)・fake Dispatcher・記録Frontend。
- GREEN: Run 実装（F1/F2）、conversation 整形、specBuilder。
### [x] Step 4: コード要約md（code-summary/test-summary）
### [x] Step 5: ローカル検証（build/test -race/gofmt/vet）

## ストーリートレーサビリティ
| Story | ステップ | 完了条件 |
|---|---|---|
| US-3.1 | 3 | ツール→完了で複数ステップ実行・変更集約 |
| US-3.2 | 3 | OnStep 通知 |
| US-3.3 | 3 | maxSteps 停止（PBT） |
