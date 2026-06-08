# Functional Design Plan — U2 LLM Connectivity（CONSTRUCTION / Part 1）

**Unit**: U2 LLM Connectivity（C3 LLM Client）
**担当ストーリー**: US-2.1（指定先/モデルへ送信）, US-2.2（SSEストリーミング）, US-6.1（接続失敗の分かりやすい案内）。
**依存**: U1（Config: endpoint/model 等、Logger）。
**規約**: TDD（test-first）+ PBT(rapid)。技術非依存でビジネスロジック中心（HTTP実装詳細はCode Generation）。
**設計前提（Application Design）**: ハイブリッドなツール呼び出し（Q2=C: function calling対応なら使用、非対応はJSON/ReActフォールバック）。SSEストリーミング。プロバイダはLM Studio固定。

> 各 `[Answer]:` に記入。「おまかせ」で推奨採用。

## 1. 生成プラン（Part 2で実施）
- [ ] `construction/U2-llm/functional-design/domain-entities.md`（Message/Request/Stream/Chunk/ToolCall/Caps/LLMError）
- [ ] `construction/U2-llm/functional-design/business-rules.md`（リクエスト組立・SSEパース・ハイブリッド判定・エラー分類。テスト可能な表明 + PBT候補）
- [ ] `construction/U2-llm/functional-design/business-logic-model.md`（Completeフロー・ストリーム処理・フォールバック・エラーマッピング）
- [ ] 設計整合性検証（Security Baseline関連の反映確認）

---

## 2. 確認質問

### Q1. function calling 対応の判定方法（ハイブリッドの肝, Q2=C）
モデルがネイティブ function calling 対応かをどう決めるか。
- **A**: **設定 `toolMode: auto|function|json`（既定 auto）**。auto は「まず function calling を試し、ツール未呼び出し/エラー/未対応応答ならJSONフォールバック」【推奨】
- **B**: 常に JSON プロンプト方式（function callingは使わない＝最も堅牢・モデル非依存）
- **C**: 起動時に1回プローブして判定しキャッシュ
- **D**: その他

[Answer]:

---

### Q2. JSONフォールバック時のツール呼び出しフォーマット
function calling非対応時、LLMにどう「ツールを呼べ」と指示し、どう解釈するか。
- **A**: **単一JSONオブジェクト**を出力させる（例: `{"tool":"name","args":{...}}` または `{"final":"..."}`）。1ステップ1アクション、パースが厳密【推奨】
- **B**: ReAct形式（`Thought:/Action:/Action Input:/Observation:`）をパース
- **C**: その他

[Answer]:

---

### Q3. 会話メッセージモデル
- **A**: OpenAI互換の roles（`system`/`user`/`assistant`/`tool`）をそのまま採用。system はエージェント方針（U4で注入）、tool は実行結果【推奨】
- **B**: 独自簡略モデル
- **C**: その他

[Answer]:

---

### Q4. ストリーミングのチャンク種別
SSEのdeltaをどう上位（U4/U5）へ渡すか。
- **A**: **Chunk を種別付き**で返す（`TextDelta` / `ToolCallDelta` / `Done`）。呼び出し側はテキスト逐次表示とツール呼び出し蓄積を区別できる【推奨】
- **B**: テキストのみ返す（ツール呼び出しは完了後にまとめて）
- **C**: その他

[Answer]:

---

### Q5. 接続・実行エラーの分類とユーザー向け文言（US-6.1, SECURITY-09）
- **A**: **分類**: 接続不可(refused/DNS) / タイムアウト / HTTP4xx-5xx / SSE破損 / モデル不存在。各に原因+対処（起動状態/URL/モデル名）を示す一般化メッセージ。内部詳細・スタックは出さない【推奨】
- **B**: 単純化（成否のみ）
- **C**: その他

[Answer]:

---

### Q6. リトライ方針
- **A**: **接続不可/タイムアウト/5xx は短い指数バックオフで少回数（例: 2回）リトライ**。4xx・モデル不存在は即時失敗（リトライ無意味）。リトライ可否/回数は設定可【推奨】
- **B**: リトライしない（MVP簡素）
- **C**: その他

[Answer]:

---

### Q7. リクエストパラメータの扱い
temperature/top_p/max_tokens 等。
- **A**: **最小限**（必要なら temperature と max_tokens のみ設定可、既定はモデル任せ/控えめ）。それ以外はMVPで固定/省略【推奨】
- **B**: 主要パラメータを一通り設定可能に
- **C**: その他

[Answer]:

---

### Q8. その他（任意）
タイムアウト既定値、リクエスト本文のログ方針（マスキング前提）、参考にしたいOpenAI/LM Studio挙動など。

[Answer]:
