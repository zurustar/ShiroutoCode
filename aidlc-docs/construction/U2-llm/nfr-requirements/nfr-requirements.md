# NFR Requirements — U2 LLM Connectivity

**Unit**: U2。規約: TDD + PBT(rapid)、依存最小化。

## 適用判定
| 領域 | 判定 | 内容 |
|---|---|---|
| Scalability | N/A | 単一クライアント、同時リクエスト1（MVP） |
| Availability/DR | N/A | ローカル依存(LM Studio)、状態なし |
| Performance | 要件あり | NFR-P1〜P3 |
| Security | 要件あり | NFR-S1〜S2 |
| Reliability | 要件あり | NFR-R1（リトライ=Functional R7）、NFR-R2（中断） |
| Maintainability | 要件あり | NFR-M1（TDD/PBT/モックHTTP/依存最小） |
| Usability | 要件あり | NFR-U1（接続エラーの分かりやすさ=Functional R6） |

## Performance
- **NFR-P1（応答性/TTFT隠蔽）**: 最初のトークン受信後ただちに逐次表示。受信→`TextDelta`発行までのバッファリングをしない（US-2.2）。
- **NFR-P2（タイムアウト）**:
  - 接続(ダイヤル)タイムアウト: 短め（既定 5s）。
  - 応答全体タイムアウト: 長め・**設定可**（既定 300s）。ローカルモデルのロード/生成を考慮。
  - **ストリーム・アイドルタイムアウト**: 一定時間（既定 60s）新規トークンが来なければ打ち切り（`BadStream`/`Timeout`扱い）。
- **NFR-P3**: U2はI/Oバウンド。CPU最適化は不要。`context` で全I/Oをキャンセル可能に。

## Security
- **NFR-S1**（SECURITY-09）: 認証情報を既定で持たない。エラーは一般化（Functional R6）。
- **NFR-S2**（SECURITY-03, U1継承）: リクエスト/レスポンス本文のログはマスキング前提（プロンプト本文は既定要約、トークンは`***`）。
- http/https 両対応。https のサーバ証明書は標準検証（無効化しない）。

## Reliability
- **NFR-R1**（Functional R7）: retryable(Unreachable/Timeout/5xx)のみ指数バックオフで少回数（既定2）。
- **NFR-R2**（NFR-3）: `context` キャンセルで進行中の接続・ストリーム・リトライ待機を即中断。

## Maintainability
- **NFR-M1**: TDD（test-first）。`httptest.Server` でHTTP/SSEをモックし hermetic にテスト。PBTを R2/R3/R4/R5/R6/R7 に適用。依存は標準ライブラリのみ（SECURITY-10）。

## Usability
- **NFR-U1**（Functional R6, US-6.1）: 5分類のエラーに原因+対処を提示。

## テスト可能な受け入れ観点
- ストリーム受信で TextDelta が逐次（全文一括でなく）発行される（P1）。
- アイドル/全体タイムアウトが ctx 経由で機能し、超過時に適切な LLMError を返す（P2）。
- ctx キャンセルでリトライ待機・ストリームが即終了する（R2）。
- ログにトークン/プロンプト生本文が出ない（S2, PBT）。
