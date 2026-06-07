# Application Design Plan — ShiroutoCode

**Stage**: INCEPTION / Application Design（Part 1: Planning）
**目的**: 主要コンポーネントの識別・責務・インタフェース・サービス層・依存関係を確定するための計画と、設計判断に必要な確認質問。
**前提**: Go製ヘッドレスコア + 対話型CLI（CLI-first）。VSCode拡張フロントは後続フェーズ。コアはフロント非依存に設計。

> 記入方法: 各質問の `[Answer]:` の後に選択肢の記号（例 `A`）または自由記述を書いてください。**全ての `[Answer]:` が埋まるまで設計成果物の生成には進みません。**「おまかせ」で各質問の推奨デフォルトを採用します。

---

## 1. 設計プラン（実行チェックリスト）

承認後、以下の成果物を生成します（detail levelは複雑さに応じて適応）:

- [ ] `application-design/components.md` — コンポーネント定義と高レベル責務・インタフェース
- [ ] `application-design/component-methods.md` — 各コンポーネントのメソッド/関数シグネチャ（詳細ビジネスルールはFunctional Designで）
- [ ] `application-design/services.md` — サービス定義とオーケストレーションパターン
- [ ] `application-design/component-dependency.md` — 依存マトリクス・通信パターン・データフロー図
- [ ] `application-design/application-design.md` — 上記を統合した設計書
- [ ] 設計の完全性・整合性の検証（Security Baseline / PBT(rapid) 拡張の適用観点を含む）

### 想定コンポーネント構成（たたき台 — Q1で確認）
Goパッケージ構成イメージ（`cmd/` + `internal/`）:
1. **CLI Frontend**（`cmd/shiroutocode`, `internal/cli`）— 引数/REPL、逐次出力、中断(Ctrl-C)、危険操作の確認プロンプト
2. **Agent Engine**（`internal/agent`）— plan→act→observe ループ、終了条件、ステップ管理、context伝播
3. **LLM Client**（`internal/llm`）— LM Studio OpenAI互換REST、SSEストリーミング、エラー処理
4. **Tool Layer**（`internal/tools`）— File / Terminal / Git / Web の各ツール、共通Toolインタフェース
5. **Guardrail**（`internal/guardrail`）— 危険操作判定・スコープ検証・フェイルクローズ（横断適用）
6. **Config**（`internal/config`）— 設定ファイル / フラグ / 環境変数の統合
7. **Observability/Logging**（`internal/log`）— 構造化ログ、機微情報マスキング

---

## 2. 確認質問

### Q1. コンポーネント分割の粒度
上記「想定コンポーネント構成」（7コンポーネント / Goパッケージ）でよいですか？
- **A**: このままでよい（7コンポーネント）
- **B**: もっと粗く統合したい
- **C**: もっと細かく分けたい
- **D**: その他（自由記述）

[Answer]:

---

### Q2. LLMのツール呼び出し方式（最重要）
ローカルLLM（LM Studio）でエージェントがツールを選ぶ仕組み。モデルによってネイティブのfunction/tool callingサポートが異なります。
- **A**: OpenAI互換の `tools`（function calling）を使う（対応モデル前提・実装は素直だがモデル依存）
- **B**: プロンプトベースのReAct/JSON出力をパースする（モデル非依存・堅牢だが実装/検証コスト高）
- **C**: ハイブリッド（function calling対応なら使い、非対応ならJSONパースにフォールバック）【推奨】
- **D**: その他

[Answer]:

---

### Q3. ファイル編集の適用方式
FR-4.2 のファイル変更をどう適用するか（出力可視化・安全性に影響）。
- **A**: ファイル全体の書き換え（full content replace、シンプル）
- **B**: 差分/パッチ適用（diff/patch、変更箇所が明確だが実装複雑）
- **C**: 両対応（小変更はパッチ、大変更は全書き換え）【推奨】
- **D**: その他

[Answer]:

---

### Q4. ターミナル実行の方式
FR-4.3 のコマンド実行（Go `os/exec`）。出力の扱い。
- **A**: 出力を完全キャプチャしてから表示
- **B**: 標準出力/標準エラーをストリーム表示（逐次・長時間コマンドに強い）【推奨】
- **C**: その他

[Answer]:

---

### Q5. ガードレール（安全制御）の適用ポイント
US-5.2/5.3、SECURITY-11 の中核。どこで危険判定を強制するか。
- **A**: 全ツール呼び出しを通す単一インターセプタ層で集中チェック（バイパス不可）【推奨】
- **B**: 各ツール内で個別にチェック
- **C**: その他

[Answer]:

---

### Q6. 会話・エージェント状態の永続化
ペルソナP1は複数PCを行き来します。会話履歴/エージェント状態の扱いは？
- **A**: セッション内のメモリのみ（プロセス終了で消える、最小実装）【推奨（MVP）】
- **B**: ワークスペース内に永続化（例: `.shiroutocode/` 配下、リポジトリ経由で持ち運び可）
- **C**: ホーム配下（`~/.config/shiroutocode` 等）に保存
- **D**: その他

[Answer]:

---

### Q7. アーキテクチャ/設計スタイル
- **A**: レイヤード（CLI / Application(エージェント) / Domain(ガードレール・ツール抽象) / Infrastructure(LLM・OS・Git)）でクリーンに分離。Goの `internal/` でパッケージ境界を強制（テスト容易・推奨）【推奨】
- **B**: シンプルなパッケージ分割（過度な抽象化を避ける）
- **C**: その他

[Answer]:

---

### Q8. CLIの操作モデル
FR-1。どの操作形態を優先しますか？
- **A**: 対話REPL優先（起動後プロンプトで連続指示）
- **B**: 単発実行優先（`shiroutocode "指示"` 一回完結）
- **C**: 両方を最初から同等にサポート【推奨】
- **D**: その他

[Answer]:

---

### Q9. その他の制約・要望（任意）
モジュールパス（例 `github.com/zurustar/shiroutocode`）、CLIフレームワークの好み（標準`flag` / `cobra` / `urfave/cli`）、バイナリ名、避けたい依存、参考にしたい既存ツール（Claude Code / aider 等）など、あれば記述してください。

[Answer]:
