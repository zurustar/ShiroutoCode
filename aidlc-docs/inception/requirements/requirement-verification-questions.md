# Requirements Clarification Questions — ShiroutoCode

各質問について、`[Answer]:` タグの後ろに選択した記号（A, B, C ...）を記入してください。
どれも当てはまらない場合は最後の「Other」を選び、`[Answer]:` の後に内容を記述してください。
複数選択が適切な質問には「（複数選択可）」と記載しています。
すべて記入したら「done」「完了」などとお知らせください。

---

## Question 1: ツールの中核的な役割
このツールは「AI駆動開発」として、主に何を実現したいですか？（最も近いものを1つ）

A) 対話型アシスタント — チャットで質問・コード提案を受け、適用は人間が手動で行う
B) 自律エージェント — 指示を与えると、ファイル編集・コマンド実行などを自動で多段階に実行する（Claude Code / Cursor Agent 的）
C) 上記の両方 — 対話モードと自律エージェントモードを切り替えられる
X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 2: エージェントが実行できる操作（複数選択可）
（Q1でB/Cを選んだ場合）AIに許可したい操作はどれですか？該当する記号をすべて記入してください。

A) ワークスペース内のファイル読み取り
B) ファイルの新規作成・編集・削除
C) ターミナルコマンドの実行（ビルド・テスト等）
D) Webアクセス・外部情報取得
E) Git操作（コミット・ブランチ作成等）
X) Other (please describe after [Answer]: tag below)

[Answer]:ACDE。Bがワークスペース内のファイルを対象としているのであればBも。


---

## Question 3: 操作の承認フロー
AIがファイル変更やコマンド実行を行う際の承認方法は？

A) 都度承認 — 各操作の前に人間が承認する（安全重視）
B) 自動承認 — AIが自律的に実行し、人間は結果を確認（速度重視）
C) 設定で切替可能 — 操作種別ごとに承認ポリシーを設定できる
X) Other (please describe after [Answer]: tag below)

[Answer]: 基本的に自動承認にしたいが、システムの破壊や外部への攻撃など、実施すべきでないことはやらないで欲しい。

---

## Question 4: VSCode拡張のUI形態（複数選択可）
利用者に提供するUIはどれですか？該当する記号をすべて記入してください。

A) サイドバーのチャットパネル（Webview）
B) インライン補完／コードアクション（エディタ内提案）
C) コマンドパレットからの実行
D) 差分（Diff）プレビューでの変更確認UI
X) Other (please describe after [Answer]: tag below)

[Answer]: claude codeみたいなやつ、Aかな？

---

## Question 5: LMStudio との接続方式
LMStudio はローカルでOpenAI互換のAPIサーバ（既定 http://localhost:1234/v1）を提供できます。接続方式の想定は？

A) OpenAI互換 REST API（/v1/chat/completions）を直接呼び出す
B) LM Studio公式SDK（@lmstudio/sdk 等）を利用する
C) どちらでもよい／おすすめに従う
X) Other (please describe after [Answer]: tag below)

[Answer]: できることが同じならA

---

## Question 6: モデル設定の柔軟性
利用するモデルやエンドポイントの設定は？

A) 設定画面でエンドポイントURL・モデル名を自由に指定できる
B) LMStudio固定（ローカルのみ）でシンプルに
C) LMStudioに加え、将来的に他プロバイダ（OpenAI/Anthropic等）も追加できる拡張性を持たせる
X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 7: 実装言語
VSCode拡張の実装言語は？（VSCode拡張は通常TypeScript/JavaScript）

A) TypeScript（推奨・型安全）
B) JavaScript
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 8: MVP（最初に動かす最小範囲）のゴール
最初のリリースで「動いた」と言える最小ゴールは？

A) サイドバーのチャットでLMStudioのローカルLLMと会話でき、応答が表示される
B) チャットに加え、AIの提案をワンクリックでファイルに適用できる
C) 指示を与えると複数ファイルを自動編集する簡単なエージェントループが動く
X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 9: 想定利用者
主な想定ユーザーは？

A) 自分自身（個人ツール）
B) チーム内での共有利用
C) 一般公開（VS Code Marketplace等への配布）
X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question: Security Extensions
このプロジェクトでセキュリティ拡張ルールを適用しますか？

A) Yes — すべてのSECURITYルールをブロッキング制約として適用する（本番品質のアプリに推奨）
B) No — SECURITYルールをすべてスキップする（PoC・プロトタイプ・実験的プロジェクトに適する）
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question: Property-Based Testing Extension
このプロジェクトでプロパティベーステスト（PBT）ルールを適用しますか？

A) Yes — すべてのPBTルールをブロッキング制約として適用する（ビジネスロジック・データ変換・シリアライズ・状態を持つコンポーネントがあるプロジェクトに推奨）
B) Partial — 純粋関数とシリアライズのラウンドトリップに対してのみPBTルールを適用する（アルゴリズム的複雑性が限定的なプロジェクトに適する）
C) No — PBTルールをすべてスキップする（単純なCRUD、UIのみ、ロジックの薄い連携層に適する）
X) Other (please describe after [Answer]: tag below)

[Answer]: A
