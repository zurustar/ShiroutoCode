# Functional Design Plan — U1 Foundation（CONSTRUCTION / Part 1 Planning）

**Unit**: U1 Foundation（C6 Config, C7 Logging）
**担当ストーリー**: US-2.1（設定）主、US-3.3/5.3/6.1 へ設定・ワークスペースルートを提供、横断: 機微情報マスキング。
**開発規約**: **TDD（test-first: red→green→refactor）** + 単体テスト + PBT（rapid）。本設計ではビジネスルールを**テスト可能な表明**として記述する。
**技術非依存**: 本ステージはビジネスロジック/ドメインモデル/ルールに集中（実装詳細・依存ライブラリはCode Generationで）。

> 記入方法: 各 `[Answer]:` に記号または自由記述。**全て埋まるまで生成（Part 2）に進みません。**「おまかせ」で推奨デフォルト採用。

---

## 1. 生成プラン（Part 2で実施）
- [x] `construction/U1-foundation/functional-design/domain-entities.md` — Config / GuardrailPolicy / LogRecord 等のドメインモデル
- [x] `construction/U1-foundation/functional-design/business-rules.md` — 設定の優先順位・検証ルール、マスキングルール（テスト可能な表明 + PBTプロパティ候補）
- [x] `construction/U1-foundation/functional-design/business-logic-model.md` — Load フロー / マスキングフロー / エラー処理
- [x] 設計の整合性検証（Security Baseline 関連ルールの反映確認）

---

## 2. 確認質問

### Q1. 設定項目（Configのフィールド）
MVPのConfigに含める項目はこれでよいですか？
`endpoint`(既定 http://localhost:1234/v1), `model`(必須), `maxSteps`(既定値あり), `workspace`(既定=カレント), `guardrail`(挙動: 確認モード/denylist調整), `logLevel`(既定 info)
- **A**: これでよい【推奨】
- **B**: 追加/削除したい（自由記述）

[Answer]: A（おまかせ）

---

### Q2. 設定ファイルの形式と場所
- **A**: **YAML**、プロジェクト配下 `.shiroutocode.yaml` ＋ ホーム `~/.config/shiroutocode/config.yaml`（プロジェクト優先）【推奨】
- **B**: JSON
- **C**: TOML
- **D**: その他/場所変更（自由記述）

[Answer]: A（おまかせ）

---

### Q3. 設定の優先順位（確定）
`フラグ > 環境変数 > プロジェクト設定ファイル > ホーム設定ファイル > 既定` でよいですか？（環境変数prefix例: `SHIROUTO_`）
- **A**: これでよい【推奨】
- **B**: 変更したい（自由記述）

[Answer]: A（おまかせ）

---

### Q4. 必須項目欠如・検証失敗時の挙動
`model` 未設定や `endpoint` のURL不正、`maxSteps<=0` 等のとき:
- **A**: 起動時に**即エラー終了**し、どの項目がなぜ不正かを一般化メッセージで提示（フェイルクローズ, SECURITY-09）【推奨】
- **B**: 既定値で続行できるものは続行し警告
- **C**: その他

[Answer]: A（おまかせ）

---

### Q5. マスキング対象（ログ機微情報, SECURITY-03）
構造化ログで必ずマスクすべきものは？
- **A**: APIトークン/認証ヘッダ、設定ファイル中のsecret、LLMへ送る/から受けるプロンプト本文の既定マスク（必要時のみdebugで全文）【推奨】
- **B**: トークン/認証情報のみ（プロンプト本文はマスクしない）
- **C**: その他（自由記述）

[Answer]: A（おまかせ）

---

### Q6. ログ出力先・形式
- **A**: 既定は**stderrへ人間可読**、`--log-format=json` で構造化JSON、`--log-file` でファイル出力可【推奨】
- **B**: 常にJSON
- **C**: その他

[Answer]: A（おまかせ）

---

### Q7. その他（任意）
U1の設定/ログに関する追加要望（環境変数prefix名、設定キー命名、相関ID付与方針など）。

[Answer]: 特になし（おまかせ）
