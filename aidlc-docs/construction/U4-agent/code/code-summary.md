# Code Summary — U4 Agent Engine

**生成日**: 2026-06-10 / TDD + PBT(rapid)。**全テスト green / -race クリーン**。新規外部依存なし。

## 生成/変更ファイル
| パス | 種別 | 内容 |
|---|---|---|
| `internal/llm/sse.go` | 変更 | `CollectStreaming(stream,onText)` 追加、`Collect` を委譲に refactor |
| `internal/agent/agent.go` | 新規 | `Frontend`/`NoopFrontend`/`Dispatcher` iface, `Runner`, `Result`/`Status`, `Run`(F1), DefaultSystemPrompt |
| `internal/agent/conversation.go` | 新規 | `specs`(F3), `appendAssistant`/`appendObservation`(R6) |
| `internal/agent/agent_test.go` | 新規 | 6テスト（PBT 1: 停止性） |

## 設計対応
| ルール | 実装 | テスト |
|---|---|---|
| R1 ループ/逐次転送 | Run + CollectStreaming(OnAssistantText) | TestSingleShot/ToolThenComplete |
| R2 完了判定 | ツール無し→Completed(Summary) | TestSingleShotCompletes |
| R3 最大ステップ(停止性) | for step<=maxSteps | TestMaxStepsTerminationPBT |
| R4 中断 | ループ頭で ctx.Err()→Aborted | TestCancelAborts |
| R5 Dispatcher経由 | disp.Dispatch のみ | TestToolThenComplete |
| R6 観測/ブロック継続 | appendObservation（err→観測化） | TestBlockedToolContinues |
| R7 空プロンプト | 先頭で拒否 | TestEmptyPromptFails |

## 拡張コンプライアンス（U4 Code）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-11 | ✔ | ツールは U3 Dispatcher 経由のみ |
| SECURITY-15 | ✔ | フェイルクローズ（Failed/Aborted）、停止性保証 |
| PBT-09 | ✔ | 停止性をPBT |
| SECURITY-03 | 継承(U1) | ログマスキング |
| SECURITY-10 | ✔ | 新規依存ゼロ |

## 注記
- 会話履歴はメモリのみ（App Q6=A）。
- Frontend Port により U5（TUI）は描画のみ実装すればよい。実LLM/実ツールのE2EはU5/Build and Test。
