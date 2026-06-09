# Code Summary — U2 LLM Connectivity

**生成日**: 2026-06-09 / **規約**: TDD + PBT(rapid)。**全テスト green / race クリーン**。**新規外部依存なし**（標準ライブラリのみ）。

## 生成ファイル（`internal/llm/`）
| パス | 種別 | 内容 |
|---|---|---|
| `types.go` | 新規 | Message/ToolSpec/ToolCall/Request/Chunk/Stream/Caps/CompletionResult/LLMClient |
| `errors.go` | 新規 | `LLMError`/`ErrorKind`, `classifyError`(P4), `classifyCtx` |
| `sse.go` | 新規 | `sseReader`(P3), `parseStreamData`, `assembleToolCalls`(R5), `streamImpl`(idle P1), `Collect` |
| `jsonfallback.go` | 新規 | `parseJSONTool`(R3), `fallbackSystemPrompt`, `extractJSONObject` |
| `client.go` | 新規 | `Client`/Options, `Complete`, `resolveMode`(R2/P5), `doStreaming`(retry P2), `buildBody`(R1) |
| `*_test.go` | 新規 | 17テスト（うちPBT 4） |

## 設計対応
| ルール/パターン | 実装 |
|---|---|
| R1 リクエスト組立（tools送信条件/param省略） | `buildBody`, `TestRequestToolsOnlyInFunctionMode` |
| R2/P5 ハイブリッド・モード解決/フォールバック | `resolveMode`/`Complete`, `TestAutoFallbackToJSON` |
| R3 JSONフォールバック規約 | `parseJSONTool`, `TestJSONToolRoundTripPBT` |
| R4 SSEパース・text保存則 | `sseReader`/`parseStreamData`, `TestStreamTextReconstructionPBT` |
| R5 tool_call断片結合 | `assembleToolCalls`, `TestToolCallAssemblyPBT` |
| R6 エラー分類・文言 | `classifyError`, `TestClassify*`, `TestUserMessageNoLeakPBT` |
| R7/P2 リトライ | `doStreaming`+`backoff`, `TestRetryOn5xx*`/`TestNoRetryOn4xx` |
| P1 タイムアウト/中断 | `streamImpl`(idle), ctx伝播, `TestContextCancelAborts` |

## 拡張コンプライアンス（U2 Code）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 | ✔ | 応答JSON/SSE検証、未知応答はDecode/BadStream |
| SECURITY-09 | ✔ | `UserMessage`一般化（PBTで非漏洩検証）、wrappedは内部のみ |
| SECURITY-13 | ✔ | LLM出力の厳密JSON解釈、不能時は実行せず停止 |
| SECURITY-15 | ✔ | フェイルクローズ、ctx中断、retryable限定 |
| PBT-09 (rapid) | ✔ | R3/R4/R5/R6 をPBT化 |
| SECURITY-10 | ✔ | 新規依存ゼロ（net/http, bufio, encoding/json, httptest） |
| SECURITY-03 | 継承(U1) | ログはマスキングLogger前提（idle/retryログにトークン非出力） |
| SECURITY-11 | N/A(U3) | ガードレールはU3 |

## 注記
- `Stream` に `Mode()` を追加（json/function）。Collectが json モード時に単一JSONフォールバックを解釈（domain-entitiesの設計を実装で具体化）。
- 実 LM Studio 接続のE2EはU5完成時。本unitは `httptest` で全経路 green。
