# Unit of Work Plan — ShiroutoCode（Units Generation: Part 1 Planning）

**前提**: Go製ヘッドレスコア + 対話型CLI（CLI-first）。**配布は単一バイナリ＝デプロイ単位は1つ（モノリス）**。よって「Service（独立デプロイ単位）」は1つで、論理的な「Module」と開発用の「Unit of Work（ストーリーの束）」をどう切るかが本ステージの主眼。

> 記入方法: 各質問の `[Answer]:` に選択肢の記号または自由記述を。**全て埋まるまで生成（Part 2）に進みません。**「おまかせ」で推奨デフォルトを採用します。

---

## 1. 生成プラン（実行チェックリスト／Part 2で実施）
- [ ] `application-design/unit-of-work.md` — unit定義・責務・コード組織戦略（Greenfield）
- [ ] `application-design/unit-of-work-dependency.md` — unit間依存マトリクスと実装順序
- [ ] `application-design/unit-of-work-story-map.md` — 全ストーリー(US-*)のunit割当
- [ ] unit境界・依存の検証、全ストーリーが必ずいずれかのunitに割当済みであることの確認

---

## 2. たたき台：unit分割案（Q1で確認）
Application Design の7コンポーネントと build sequence を踏まえた **5 units（漸進的）案**:

| Unit | 含むコンポーネント | 主なストーリー |
|---|---|---|
| **U1 Foundation** | C6 Config, C7 Logging | US-2.1(設定), US-6.1(一部) |
| **U2 LLM Connectivity** | C3 LLM Client | US-2.1, US-2.2, US-6.1 |
| **U3 Tools & Guardrail** | C4 Tool Layer, C5 Guardrail | US-4.1〜4.5, US-5.1〜5.3, US-6.2 |
| **U4 Agent Engine** | C2 Agent Engine | US-3.1〜3.3 |
| **U5 CLI Frontend** | C1 CLI（Frontend Port実装） | US-1.1〜1.3, US-3.2(表示), 危険操作確認UI |

依存（実装順）: U1 → U2 → U3 → U4 → U5（U4はU2,U3に依存、U5は全体を統合）。

---

## 3. 確認質問

### Q1. Unit分割の方針
Construction フェーズは **unit単位でループ**（各unitごとに Functional/NFR/Code Generation）します。どう切りますか？
- **A**: 上記 **5 units（漸進的）**【推奨】— 各層を順に作り込み、早期に部分テスト可能
- **B**: **単一unit**（CLI全体を1 unit）— Construction 1パスで一括。最小オーバーヘッドだが粒度が粗い
- **C**: 別の粒度（例: U3 を Tools と Guardrail に分ける等）→ 自由記述で指定
- **D**: その他

[Answer]:

---

### Q2. 実装順序（漸進案を採る場合）
U1→U2→U3→U4→U5 の順でよいですか？（依存的に妥当な順）
- **A**: この順でよい【推奨】
- **B**: 別の順序を希望（自由記述）
- **C**: Q1でBを選んだので該当なし

[Answer]:

---

### Q3. 各unit完成の「動作確認」境界
unitごとにどこまで動けば「完了」とみなしますか？
- **A**: 各unitは**単体テスト/PBT green**で完了。end-to-end動作はU5完成時（最小オーバーヘッド）【推奨】
- **B**: 各unit完了時に、その時点で動く範囲のCLI実行も確認する（モック等で繋ぐ手間あり）
- **C**: その他

[Answer]:

---

### Q4. コード組織（Greenfield・確認）
Application Design で定めた `cmd/shiroutocode` + `internal/{cli,agent,llm,tools/...,guardrail,config,log}`（単一モジュール・単一バイナリ）でよいですか？
- **A**: これでよい【推奨】
- **B**: 変更したい（自由記述：モジュールパス/ディレクトリ等）

[Answer]:

---

### Q5. その他（任意）
unit分割・順序・命名・テスト方針について追加の要望があれば記述してください。

[Answer]:
