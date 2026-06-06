# Story Generation Plan — ShiroutoCode

このファイルは User Stories ステージ Part 1（計画）の成果物です。
下部の質問について `[Answer]:` タグの後ろに選択記号（A, B, C ...）を記入してください。
どれも当てはまらない場合は「Other」を選び内容を記述してください。記入後「done」とお知らせください。

---

## 1. アプローチ（Part 2 で実行する手順）
- [x] 承認された方針に基づき `personas.md`（ユーザー像）を生成
- [x] `stories.md` を生成（INVEST準拠: Independent, Negotiable, Valuable, Estimable, Small, Testable）
- [x] 各ストーリーに受け入れ基準（Acceptance Criteria）を付与
- [x] ペルソナと各ストーリーをマッピング
- [x] セーフティ/ガードレール、エラー時挙動を受け入れ基準として明文化（Security/PBT拡張の入力）

## 2. ストーリーのスコープ（要件からの導出・暫定）
要件 (requirements.md) から、以下のエピック単位を想定:
- E1: チャットUI（サイドバーWebview）での対話
- E2: LM Studio接続・設定・ストリーミング応答
- E3: 自律エージェントループ（複数ファイル自動編集）
- E4: ツール実行（ファイル/ターミナル/Git/Web）と可視化
- E5: 承認フロー & セーフティガードレール
- E6: エラー処理・接続失敗時のUX

---

# 質問（Part 1）

## Question 1: ストーリーの分割アプローチ
ユーザーストーリーをどの観点で整理しますか？

A) Epic-Based — 上記E1〜E6のエピック配下にサブストーリーをぶら下げる階層構造
B) User Journey-Based — 「指示を出して完了するまで」の利用フローに沿って並べる
C) Feature-Based — 機能（チャット/接続/ループ/ツール/ガードレール）単位で整理
D) Hybrid — Epicで束ねつつ各Epic内はUser Journey順に並べる
X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 2: ペルソナの範囲
どのユーザー像をペルソナとして定義しますか？（最も近いもの1つ）

A) エンドユーザー（拡張を使う開発者）のみ
B) エンドユーザー + 拡張の保守者（contributor/maintainer）
C) エンドユーザーを利用習熟度で分割（初心者／熟練者）+ 保守者
X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 3: 受け入れ基準（Acceptance Criteria）の形式
受け入れ基準の記述形式は？

A) Given-When-Then（Gherkin風、テスト・PBTに紐づけやすい）
B) チェックリスト形式（箇条書きの満たすべき条件）
C) 両方併用（重要シナリオはGiven-When-Then、補助条件はチェックリスト）
X) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 4: ストーリーの粒度
ストーリーの大きさの方針は？

A) 細かめ — 1ストーリー=1機能の小単位（数が増えるが見積もり/テストしやすい）
B) 中程度 — 利用者にとって意味のある単位でまとめる
C) おまかせ（INVESTに従い適切に判断）
X) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 5: MVPの明示
MVP（Q8=C: 複数ファイル自動編集が動く）に必要なストーリーを区別しますか？

A) Yes — 各ストーリーに「MVP対象/対象外」のタグを付ける
B) No — 区別せず全ストーリーをフラットに列挙する
X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 6: 記述言語
ストーリー/ペルソナの記述言語は？

A) 日本語（README/要件に合わせる）
B) 英語
C) 日本語主体・技術用語は英語併記
X) Other (please describe after [Answer]: tag below)

[Answer]: A
