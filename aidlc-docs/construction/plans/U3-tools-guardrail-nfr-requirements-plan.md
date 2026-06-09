# NFR Requirements Plan — U3 Tools & Guardrail（CONSTRUCTION / Part 1）

**Unit**: U3。規約: TDD + PBT(rapid)、依存最小化。
**早見**: Scalability/Availability=N/A。**Security が支配的NFR**（ガードレール＝中核、大半は Functional R1-R10 で定義済。本ステージで網羅性を確認）。Performance/Reliability/Maintainability は軽〜中。

## 1. 生成プラン（Part 2で実施）
- [x] `construction/U3-tools-guardrail/nfr-requirements/nfr-requirements.md`
- [x] `construction/U3-tools-guardrail/nfr-requirements/tech-stack-decisions.md`

---

## 2. 確認質問

### Q1. ターミナル実行の技術（FR-4.3）
- **A**: **標準 `os/exec`。既定は引数配列（非shell）。shell連結(`|`,`;`,`&&`)が必要なコマンドは `sh -c` で実行しつつ、denylistはコマンド文字列全体に適用**（依存なし）【推奨】
- **B**: 常に `sh -c`
- **C**: その他

[Answer]: A（おまかせ）

---

### Q2. ターミナルのタイムアウト/リソース
- **A**: **コマンドに ctx 連動の実行タイムアウト（既定 例 120s, 設定可）。ctxキャンセルでプロセスツリー終了。出力サイズ上限で打ち切り**【推奨】
- **B**: タイムアウトなし（MVP簡素）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q3. Git/Web の実装
- **A**: **Git=`git` CLI を `os/exec`（依存なし、ユーザー環境のgit利用）。Web=標準 `net/http`（GET、サイズ上限、リダイレクト制限）**【推奨】
- **B**: go-git 等のライブラリ（依存増）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q4. ガードレール denylist の管理
- **A**: **コード内蔵の既定ルール表（データ駆動）＋ Config `ExtraDenyPatterns` で追加。将来の更新が容易な構造**【推奨】
- **B**: ハードコードのみ
- **C**: その他

[Answer]: A（おまかせ）

---

### Q5. パフォーマンス
- **A**: **判定はO(ルール数)で軽量、実行前に都度評価。ファイル/コマンドI/OがボトルネックでありMVPで追加最適化なし**【推奨】
- **B**: その他

[Answer]: A（おまかせ）

---

### Q6. その他（任意）
出力サイズ上限値、HTTPリダイレクト最大数、git実行のenv（認証）方針など。

[Answer]: 特になし（おまかせ）
