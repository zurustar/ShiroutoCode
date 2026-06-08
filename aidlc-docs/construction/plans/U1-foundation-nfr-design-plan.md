# NFR Design Plan — U1 Foundation（CONSTRUCTION / Part 1 Planning）

**Unit**: U1 Foundation。規約: TDD + PBT(rapid)、依存最小化。
**カテゴリ適用判定**:
- Resilience: フェイルクローズ（決定済 R4/NFR-R1）→ パターン化のみ
- Scalability: **N/A**（単一プロセス）
- Performance: 同期ログ・ロード即時（決定済）→ 軽微
- Security: マスキング/検証パターン → **主論点**
- Logical Components: 外部インフラ部品（queue/cache/circuit breaker）は **無し（N/A）**

> 質問は実質的な設計選択のみ。`[Answer]:` 記入。「おまかせ」で推奨採用。

## 1. 生成プラン（Part 2で実施）
- [ ] `construction/U1-foundation/nfr-design/nfr-design-patterns.md`
- [ ] `construction/U1-foundation/nfr-design/logical-components.md`

---

## 2. 確認質問

### Q1. ログ・マスキングの実装パターン
SECURITY-03（NFR-S3, R6）の実現方法。
- **A**: **slog.Handler デコレータ**（基底Handlerをラップし、出力直前に全属性へMaskRuleを適用）。呼び出し側はマスクを意識不要・抜け漏れにくい【推奨】
- **B**: 各ログ呼び出し側でマスクしてから渡す（実装単純だが漏れやすい）
- **C**: その他

[Answer]:

---

### Q2. 設定検証エラーの集約パターン
R4「複数不正をまとめて提示」の実装。
- **A**: **エラー集約**（`errors.Join` で全違反を集めて1度に返す）【推奨】
- **B**: 最初の違反で即時return（実装単純だがUX劣る）
- **C**: その他

[Answer]:

---

### Q3. 設定マージの実装パターン
- **A**: **段階的上書き**（default構造体に各ソースを順に重ね、"設定済みか"を区別して優先順位適用）【推奨】
- **B**: その他（自由記述）

[Answer]:

---

### Q4. その他（任意）
リトライ等の追加パターン要望（U1にHTTP等は無いため通常不要）。

[Answer]:
