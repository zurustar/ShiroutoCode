# Code Generation Plan — U2 LLM Connectivity（CONSTRUCTION / Part 1）

**Unit**: U2 LLM Connectivity（`internal/llm`）
**規約**: **TDD（test-first）** + PBT(rapid)。本plan が単一の真実。
**ワークスペースルート**: `/Users/oumi/Documents/GitHub/ShiroutoCode`。
**依存**: U1（`internal/config`, `internal/log`）。新規外部依存なし（標準ライブラリのみ、test は既存 rapid + 標準 httptest）。

## 担当ストーリー
- US-2.1（指定先/モデルへ送信）, US-2.2（SSEストリーミング逐次）, US-6.1（接続失敗の案内）

## コード配置（確定）
```
internal/llm/
├── types.go        # Message, Request, ToolSpec, ToolCall, Chunk, Caps, CompletionResult, Stream, Client(interface)
├── errors.go       # LLMError, ErrorKind, classifyError (P4)
├── sse.go          # sseReader (P3), streamImpl (Chunk変換, idleタイマ P1)
├── jsonfallback.go # jsonToolParser (Functional R3)
├── client.go       # Client実装, requestBuilder(LC2), modeResolver(LC3), retrier(LC4), Complete
├── *_test.go       # 各ファイルに対応する単体 + PBT
```
※ U2単体では `main` 無し。完了条件＝`go test ./...` green（モックHTTP）。

## 生成ステップ（TDD・順次）

### [x] Step 1: 型定義（types.go）
- ドメイン型（domain-entities準拠）と `Client`/`Stream` インタフェース。実装前の土台（コンパイル用最小）。

### [x] Step 2: エラー分類 — テスト先行→実装（errors.go）
- RED: `classifyError` の単体（refused/timeout/404/4xx/5xx/BadStream/Decode）+ PBT（UserMessageに機微情報が出ない, R6）。
- GREEN: `LLMError`, `ErrorKind`, `classifyError` 実装（P4）。

### [x] Step 3: SSE — テスト先行→実装（sse.go）
- RED: `sseReader`（data連結/コメント/空行/[DONE]）単体；`streamImpl` の Chunk 変換；**PBT**: text分割→TextDelta連結＝元文字列（R4保存則）、tool_call断片結合（R5）。
- GREEN: `sseReader`/`streamImpl` 実装（P3、idleタイマ P1）。

### [x] Step 4: JSONフォールバック — テスト先行→実装（jsonfallback.go）
- RED: **PBT** 整形JSON（tool/args or final）の往復パース（R3）；不正→Decode。
- GREEN: `jsonToolParser` 実装。

### [x] Step 5: Client・組立・リトライ・モード — テスト先行→実装（client.go）
- RED（httptestモック）:
  - リクエスト組立: functionモードのみ tools 送信、temperature/max_tokens 省略条件（R1）
  - SSEストリーミングで TextDelta 逐次（P1/US-2.2）
  - リトライ: retryable のみ・回数・ctx中断（P2/R7、PBT可）
  - autoフォールバック最大1回（P5/R2）
  - タイムアウト（全体/アイドル）発火（P1）
- GREEN: `requestBuilder`/`modeResolver`/`retrier`/`Complete` 実装。

### [x] Step 6: コード要約ドキュメント
- `aidlc-docs/construction/U2-llm/code/code-summary.md` / `test-summary.md`

### [x] Step 7: ローカル検証
- `go build ./...` + `go test ./...`（rapid含む）green を確認。`gofmt`/`go vet` クリーン。

### API/Repository/Frontend/DB/デプロイ
- **N/A**（U2にUI/DB無し。HTTPクライアントはツールでなくインフラ境界）

## ストーリートレーサビリティ
| Story | ステップ | 完了条件 |
|---|---|---|
| US-2.1 | Step 1,5 | Complete が endpoint/model へ送信（httptestで検証） |
| US-2.2 | Step 3,5 | SSEで TextDelta 逐次 green |
| US-6.1 | Step 2,5 | classifyError＋接続失敗時の UserMessage green |

## スコープ概算
- 7ステップ、`internal/llm` に5実装ファイル + 対応テスト。新規外部依存なし。
