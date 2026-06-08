# Business Logic Model — U2 LLM Connectivity

> 技術非依存のフロー。TDD前提（各フローにテスト先行）。

## F1. Complete フロー（高レベル）
入力: `Request`。出力: `Stream`（逐次Chunk）または `LLMError`。
```text
1. モード解決(R2): ToolMode と Caps から function|json を決定
2. ペイロード組立(R1/R3):
     function → messages + tools(JSON Schema)
     json     → messages（systemにJSON出力規約を付与）, tools送らない
     temperature/max_tokens は非nilのみ付与
3. HTTP POST {endpoint}/chat/completions (stream=true)
     接続/HTTPエラー → 分類(R6) → リトライ判定(R7)
4. SSE 受信ループ(R4/R5):
     data行 → TextDelta / ToolCallDelta を発行
     [DONE] → Done(FinishReason)
     破損 → LLMError{BadStream}
5. autoモードでフォールバック条件成立(R2) → Caps更新し json モードで F1 を1回やり直し
```

## F2. ストリーム消費と集約
```text
呼び出し側(U4)は Recv() ループ:
  TextDelta      → 蓄積 + (U5へ)逐次表示
  ToolCallDelta  → index毎に連結
  Done           → CompletionResult 確定（Text / ToolCalls / FinishReason）
jsonモード時: Doneまでに集めたTextを単一JSONとしてパース(R3)し ToolCall/final へ正規化
```

## F3. フォールバック（auto, R2）
```text
function試行:
  - サーバが tools 非対応(400/エラー) → json へ
  - 応答に tool_calls 無く、本文が JSON規約に合致 → json と判断
  - 正常に tool_calls or 通常テキスト → function 継続（Caps=true）
フォールバックは最大1回（無限ループ防止）。確定後は Caps をプロセス内キャッシュ。
```

## F4. リトライ（R7）
```text
attempt = 0
loop:
  err = doRequestAndStreamHeaders()   # ヘッダ確立まで
  if err == nil: break
  classify(err)                       # R6
  if !err.Retryable || attempt >= maxRetries || ctx.Done(): return err
  sleep(backoff(attempt)); attempt++  # 指数, ctxで中断可
# ストリーム本体受信中の切断は原則リトライしない（重複防止）
```

## F5. エラーマッピング（R6 / SECURITY-09）
```text
net: connection refused / no such host → Unreachable
ctx deadline / client timeout          → Timeout
HTTP 404 or body"model not found"      → ModelNotFound
HTTP 4xx                               → HTTPStatus(retryable=false)
HTTP 5xx                               → HTTPStatus(retryable=true)
SSE parse / unexpected EOF             → BadStream
JSON decode (jsonモード)               → Decode
→ それぞれ UserMessage(一般化) を付与、wrapped は内部ログのみ
```

## テスト観点（TDD: 先に書く）
| 観点 | 種別 | ルール |
|---|---|---|
| ペイロード組立（tools送信条件/パラメータ省略） | unit | R1 |
| モード解決の決定性 | PBT | R2 |
| JSON規約の往復パース | PBT | R3 |
| SSE: text連結の保存則 | PBT | R4 |
| SSE: tool_call断片の結合 | PBT | R5 |
| SSE: [DONE]/破損/コメント行 | unit | R4 |
| エラー分類（refused/timeout/404/5xx/破損） | unit | R6 |
| UserMessageに機微情報が出ない | PBT | R6 |
| リトライ回数 = retryable?configured:0、ctx中断 | unit + PBT | R7 |
| autoフォールバックが最大1回 | unit | R2/F3 |

## 実装方針メモ（Code Generation向け）
- HTTPは標準 `net/http`、SSEは `bufio.Scanner` で行読み（依存最小, SECURITY-10）。
- テスト容易性: `httptest.Server` で LM Studio応答（SSE含む）をモック。`Client` は `baseURL`/`httpClient`/`Logger` を注入可能に。
- 実LM Studio接続のE2EはU5完成時（unit-of-work Q3=A）。本unitはモックHTTPで green。

## 拡張コンプライアンス（U2 Functional Design）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 | 反映 | 応答(JSON/SSE)の検証、未知応答はDecodeエラー |
| SECURITY-09 | 反映 | エラー一般化(R6)、内部情報非露出 |
| SECURITY-13 | 反映 | LLM出力の安全な解釈（厳密JSONパース、勝手に実行しない） |
| SECURITY-15 | 反映 | フェイルクローズ（解釈不能→Decodeで停止）、ctx中断 |
| PBT-09 (rapid) | 反映 | R2/R3/R4/R5/R6/R7 をPBT化 |
| SECURITY-03 | 反映(U1継承) | リクエスト/レスポンスのログはマスキング前提 |
| SECURITY-10/11 | 一部/N/A | 依存最小(net/http)。ガードレールはU3 |
