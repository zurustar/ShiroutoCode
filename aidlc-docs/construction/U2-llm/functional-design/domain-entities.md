# Domain Entities — U2 LLM Connectivity

> 技術非依存のドメインモデル。Go型名は参考（HTTP/SSE実装詳細はCode Generation）。

## E1. Message（会話の1要素, Q3=A）
OpenAI互換 roles を採用。
| フィールド | 型 | 説明 |
|---|---|---|
| `Role` | enum(`system`/`user`/`assistant`/`tool`) | 発話者 |
| `Content` | 文字列 | 本文（tool結果も文字列で格納） |
| `ToolCallID` | 文字列 \| 空 | role=tool のとき、対応するツール呼び出しID |
| `ToolCalls` | ToolCall[] | role=assistant がツールを呼ぶとき（function calling時） |

## E2. ToolSpec（LLMへ提示するツール定義）
| フィールド | 型 | 説明 |
|---|---|---|
| `Name` | 文字列 | ツール名 |
| `Description` | 文字列 | 用途 |
| `Parameters` | JSON Schema(任意) | 引数スキーマ（function calling/JSON両用） |

## E3. ToolCall（LLMが要求したツール実行）
| フィールド | 型 | 説明 |
|---|---|---|
| `ID` | 文字列 | 呼び出し識別（functionモード時） |
| `Name` | 文字列 | ツール名 |
| `Args` | map[string]any（生JSON） | 引数 |

## E4. Request（補完要求）
| フィールド | 型 | 既定 | 説明 |
|---|---|---|---|
| `Messages` | Message[] | — | 会話履歴 |
| `Tools` | ToolSpec[] | 空 | 提示するツール（functionモード時のみ送信） |
| `Stream` | bool | true | SSEストリーミング |
| `ToolMode` | enum(`auto`/`function`/`json`) | auto | ハイブリッド制御(Q1=A) |
| `Temperature` | *float | nil(=モデル任せ) | 任意(Q7=A) |
| `MaxTokens` | *int | nil | 任意(Q7=A) |

## E5. Stream / Chunk（ストリーミング出力, Q4=A）
`Stream`: `Recv() (Chunk, error)`（`io.EOF`で終了）, `Close()`。
**Chunk**（種別付き）:
| Kind | ペイロード | 用途 |
|---|---|---|
| `TextDelta` | 文字列断片 | 逐次テキスト表示（US-2.2） |
| `ToolCallDelta` | ToolCall断片（id/name/args部分） | ツール呼び出しの蓄積 |
| `Done` | FinishReason（stop/tool_calls/length） | 完了通知 |

## E6. Caps（モデル能力, Q1=A）
| フィールド | 型 | 説明 |
|---|---|---|
| `SupportsFunctionCalling` | bool/未知 | auto判定の結果（試行後に確定） |

## E7. CompletionResult（非ストリーム集約 or ストリーム集約後）
| フィールド | 型 | 説明 |
|---|---|---|
| `Text` | 文字列 | 最終アシスタントテキスト |
| `ToolCalls` | ToolCall[] | 解釈済みツール呼び出し（function or JSON由来を正規化） |
| `FinishReason` | enum | stop/tool_calls/length |

## E8. LLMError（分類付きエラー, Q5=A）
| フィールド | 型 | 説明 |
|---|---|---|
| `Kind` | enum | `Unreachable`(refused/DNS) / `Timeout` / `HTTPStatus`(4xx/5xx) / `BadStream`(SSE破損) / `ModelNotFound` / `Decode` |
| `StatusCode` | int | HTTP時 |
| `UserMessage` | 文字列 | 一般化されたユーザー向け文言（内部情報なし, SECURITY-09） |
| `Retryable` | bool | リトライ可否（Q6: Unreachable/Timeout/5xx=true, 4xx/ModelNotFound=false） |
| `wrapped` | error | 内部用（ログのみ、ユーザーに出さない） |

## 関係
```text
Request(Messages[], Tools[]) → Client.Complete → Stream(Chunk...) → CompletionResult
Client.Complete 失敗時 → LLMError(Kind, UserMessage, Retryable)
ToolMode=auto: functionモード試行 → 不成立で json モードへ（Caps更新）
```
