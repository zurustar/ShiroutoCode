# NFR Requirements Plan — U2 LLM Connectivity（CONSTRUCTION / Part 1）

**Unit**: U2 LLM Connectivity。規約: TDD + PBT(rapid)、依存最小化(SECURITY-10)。
**早見**: Scalability=N/A（単一クライアント）。Availability/DR=N/A。主論点は **性能（ストリーミング応答性・タイムアウト）/ 信頼性（リトライ＝Functionalで定義済）/ 技術選定（HTTP・SSE）**。

## 1. 生成プラン（Part 2で実施）
- [ ] `construction/U2-llm/nfr-requirements/nfr-requirements.md`
- [ ] `construction/U2-llm/nfr-requirements/tech-stack-decisions.md`

---

## 2. 確認質問

### Q1. HTTPクライアント / SSE実装
- **A**: **標準 `net/http` + `bufio` でSSE自前パース**（追加依存なし, SECURITY-10）【推奨】
- **B**: SSE/OpenAIクライアントのサードパーティライブラリを使う（依存増）
- **C**: その他

[Answer]:

---

### Q2. タイムアウト設計
ローカルLLMは初回モデルロードやトークン生成に時間がかかる点を考慮。
- **A**: **接続タイムアウトは短め（例 5s）、応答全体タイムアウトは長め/設定可（例 既定 300s）、ストリームは「アイドルタイムアウト」（一定時間トークンが来なければ打ち切り、例 60s）**【推奨】
- **B**: 単一の全体タイムアウトのみ（シンプル）
- **C**: その他（自由記述）

[Answer]:

---

### Q3. 性能目標（応答性）
- **A**: **最初のトークンを受け取り次第ただちに逐次表示（TTFTを隠蔽）。U2はI/Oバウンドなので追加の最適化はしない**【推奨】
- **B**: バッファリング/バッチ表示を要件化
- **C**: その他

[Answer]:

---

### Q4. 接続のセキュリティ（http/https）
LM Studio は通常ローカル http。
- **A**: **http/https 両対応（既定 http://localhost）。リモートhttpsも設定で可。認証ヘッダは持たない（ローカル前提, SECURITY-09）。将来トークンが要る場合は設定経由＋ログマスク**【推奨】
- **B**: その他

[Answer]:

---

### Q5. その他（任意）
コネクション再利用(keep-alive)方針、同時リクエスト数（MVPは1）など。

[Answer]:
