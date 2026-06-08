# NFR Design Patterns — U2 LLM Connectivity

> NFR要件を実装パターンへ。技術: 標準 `net/http`/`bufio`/`encoding/json`/`context`。TDDで先行テスト。

## P1. タイムアウト3層（NFR-P2, Q1=A）
- **接続**: `http.Transport.DialContext` に `net.Dialer{Timeout: connectTimeout(5s)}`。
- **全体**: 呼び出し側からの親 `ctx`（`context.WithTimeout(overallTimeout=300s, 設定可)`）をリクエストに付与。
- **アイドル**: ストリーム受信で各 `Recv` 待機に per-read タイマ（`idleTimeout=60s`）。トークン受信ごとにリセット。超過で `LLMError{Timeout}`（or BadStream）。
- すべて `ctx` 連動でキャンセル即時伝播（NFR-R2）。
- **テスト**: httptestで遅延/無音応答を再現し各タイムアウトが発火。

## P2. リトライ（指数バックオフ, Q2=A / Functional R7）
```text
backoff(n) = base * 2^n + jitter   (base 例 200ms, 上限あり)
retry対象: ヘッダ確立まで。retryable(Unreachable/Timeout/5xx)のみ。
ストリーム開始（最初のbyte受信）後は非リトライ（重複出力防止）。
待機は select{ <-time.After(backoff): ; <-ctx.Done(): return ctx.Err() }
```
- **テスト(PBT)**: 任意のerror種別で実行リトライ数 = `retryable ? min(configured,max) : 0`。ctxキャンセルで即終了。

## P3. SSE読み取りの責務分離（Q3=A）
- `sseReader`: `bufio.Scanner`（行スキャン、長い行に備え `Buffer` 拡張）で SSE を読み、`event{data string}` を1件ずつ返す。`:`コメント行・空行を無視、複数 `data:` を連結、`[DONE]` を検出。
- `streamImpl`（`Stream`実装）: `sseReader` の event をドメイン `Chunk`（TextDelta/ToolCallDelta/Done）へ変換。tool_call断片の index 連結もここ。
- 分離により SSEパース（バイト/行）とドメイン変換を独立にテスト可能。
- **テスト(PBT)**: text分割→TextDelta連結＝元文字列（保存則, R4）。tool_call断片結合（R5）。

## P4. エラー分類の集約（Q4=A / Functional R6）
```text
classifyError(err error, status int, body []byte) LLMError
  net: errors.As(*net.OpError)/refused/DNS → Unreachable
  ctx.DeadlineExceeded / idle timer        → Timeout
  status 404 || body~"model" "not found"   → ModelNotFound (retryable=false)
  status 4xx                               → HTTPStatus(false)
  status 5xx                               → HTTPStatus(true)
  SSE/EOF異常                              → BadStream
  json decode (jsonモード)                 → Decode
  → UserMessage(一般化), Retryable, wrapped(内部ログのみ)
```
- 単一関数に集約 → 一貫した文言・テスト容易。
- **テスト**: 各入力→期待Kind/Retryable/UserMessage（機微情報なし: PBT）。

## P5. フォールバック制御（auto, Functional R2/F3）
- `modeResolver`: 現在の `Caps` と `ToolMode` から function|json を返す。auto時 functionで開始し、フォールバック条件成立で `Caps.SupportsFunctionCalling=false` をプロセス内キャッシュ。フォールバックは最大1回。
- **テスト**: function非対応応答→jsonへ1回だけ切替、以後json。

## 適用しないパターン
- サーキットブレーカ/レート制限/接続プール調整: MVP（同時1リクエスト）では **N/A**。標準keep-aliveのまま。
- キャッシュ: 応答キャッシュは持たない（毎回ライブ生成）。
