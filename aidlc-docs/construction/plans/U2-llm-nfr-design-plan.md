# NFR Design Plan — U2 LLM Connectivity（CONSTRUCTION / Part 1）

**Unit**: U2。規約: TDD + PBT(rapid)。論理インフラ部品（queue/cache/circuit breaker等）は基本N/A（resilienceはリトライ＋タイムアウトで実現）。

## 1. 生成プラン（Part 2で実施）
- [x] `construction/U2-llm/nfr-design/nfr-design-patterns.md`
- [x] `construction/U2-llm/nfr-design/logical-components.md`

---

## 2. 確認質問

### Q1. タイムアウト3層の実装パターン（NFR-P2）
- **A**: **接続=Transport DialContext、全体=親 `context.WithTimeout`、アイドル=各 Recv 待機に per-read タイマ（トークン受信ごとにリセット）**【推奨】
- **B**: 単一の全体 context のみ（簡素）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q2. リトライのバックオフ実装（NFR-R1 / Functional R7）
- **A**: **自前の指数バックオフ（base*2^n + 小jitter、ctxで中断）。ヘッダ確立まで対象、ストリーム開始後は非リトライ**【推奨, 依存なし】
- **B**: バックオフ用ライブラリを使う（依存増）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q3. SSEリーダの構造
- **A**: **`sseReader`（`bufio.Scanner`ラップ）が `event`を1件ずつyield → `streamImpl`がドメインChunkへ変換**。SSEパースとドメイン変換を分離（テスト容易）【推奨】
- **B**: 一体化（単純）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q4. エラーマッピングの実装（Functional R6）
- **A**: **`classifyError(err, statusCode, body) LLMError` の単一関数**に集約（net err種別/HTTPコード/本文ヒントから Kind と UserMessage を決定）【推奨】
- **B**: 呼び出し各所で分類
- **C**: その他

[Answer]: A（おまかせ）

---

### Q5. その他（任意）

[Answer]: 特になし（おまかせ）
