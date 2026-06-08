# Tech Stack Decisions — U2 LLM Connectivity

| # | 決定事項 | 採用 | 理由 |
|---|---|---|---|
| T1 | HTTPクライアント | **標準 `net/http`** | 追加依存なし(SECURITY-10)。タイムアウト/ctx/keep-alive対応十分 |
| T2 | SSEパース | **`bufio.Scanner`（自前）** | 軽量・依存なし。`data:`/`[DONE]`/コメント行を自前処理（Functional R4） |
| T3 | JSONエンコード/デコード | 標準 `encoding/json` | OpenAI互換ペイロード・フォールバックJSON(R3)の処理 |
| T4 | タイムアウト/キャンセル | `context` + `http.Client.Timeout`/`Transport` | 接続/全体/アイドルの3層（NFR-P2）、ctx中断（NFR-R2） |
| T5 | テスト | 標準 `testing` + `net/http/httptest` + `rapid` | TDD。SSE応答をモックサーバで再現 |

## 依存方針
- **U2で増える本体依存は無し**（標準ライブラリのみ）。テストも `httptest`（標準）+ 既存 `rapid`。
- U1で導入済みの `yaml.v3`/`rapid` 以外に新規追加なし。GPL-3.0互換は維持。

## 設計メモ（Code Generation向け）
- `Client{ baseURL, httpClient, logger, retry }` を依存注入可能に。`baseURL` は `Config.Endpoint`。
- SSEは `resp.Body` を `bufio.Scanner`（行スキャン、バッファ拡張に注意＝長い行対応）で読み、`data:` 行をJSONデコード。
- アイドルタイムアウトは「各 `Recv` 待機に ctx + time.Timer」もしくは Transport の ResponseHeaderTimeout + 読み取りデッドラインで実現（詳細はCode Generation）。
- 実 LM Studio との疎通確認は U5 完成時の E2E。
