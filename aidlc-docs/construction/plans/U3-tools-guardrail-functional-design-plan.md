# Functional Design Plan — U3 Tools & Guardrail（CONSTRUCTION / Part 1）

**Unit**: U3（C4 Tool Layer + C5 Guardrail）。最大の unit・**安全性の中核**。
**担当ストーリー**: US-4.1〜4.5（File/Terminal/Git/Web）, US-5.1〜5.3（自動承認/危険ブロック/スコープ限定）, US-6.2（フェイルクローズ）。
**依存**: U1（Config: workspace/guardrailポリシー, Logger）。
**規約**: TDD + PBT(rapid)。技術非依存。
**設計前提**: 全ツール実行は **ToolDispatcher（単一インターセプタ, App Q5=A）** を通る。編集はパッチ+全書換両対応（App Q3=C）。ターミナルはストリーム出力（App Q4=B）。

> 各 `[Answer]:` に記入。「おまかせ」で推奨採用。

## 1. 生成プラン（Part 2で実施）
- [ ] `construction/U3-tools-guardrail/functional-design/domain-entities.md`
- [ ] `construction/U3-tools-guardrail/functional-design/business-rules.md`（ツール契約・ガードレール判定・スコープ・denylist。テスト可能な表明 + PBT候補）
- [ ] `construction/U3-tools-guardrail/functional-design/business-logic-model.md`（Dispatchフロー・各ツール・確認・フェイルクローズ）
- [ ] 設計整合性検証（Security Baseline 反映確認）

---

## 2. 確認質問

### Q1. ファイル編集（FR-4.2）の指定方法（App Q3=C 具体化）
- **A**: **新規/全書換=フルcontent**、**部分編集=検索置換ブロック**（`old_string`→`new_string`、一意マッチ必須）。削除=パス指定【推奨, Claude Code/aider流】
- **B**: unified diff（patch）適用
- **C**: 常にフル書換のみ（最小）
- **D**: その他

[Answer]:

---

### Q2. ガードレール判定の出力（US-5.1/5.2）
- **A**: **`Allow` / `Confirm`(理由付き) / `Deny`(理由付き)** の3値。Confirmは確認後に実行、Denyは実行不可【推奨】
- **B**: Allow/Block の2値のみ
- **C**: その他

[Answer]:

---

### Q3. ワークスペース・スコープ限定（US-5.3, SECURITY-11）
- **A**: **対象パスを絶対化・正規化し、シンボリックリンクを解決した実体がワークスペースルート配下にあることを必須**。外れる書込/削除は Deny（読取も既定はworkspace内、外は Confirm）【推奨】
- **B**: 文字列プレフィックス一致のみ（簡易・リンク回避に弱い）
- **C**: その他

[Answer]:

---

### Q4. 危険コマンドの denylist（US-5.2）— ターミナル
既定でブロック（Deny or Confirm）にすべきパターン。
- **A**: **次を Deny: `rm -rf /`系・ワークスペース外への破壊（`rm`/`mv`の絶対パス外）・デバイス書込(`/dev/`,`dd of=`)・fork爆弾・`mkfs`・`shutdown/reboot`・パイプ実行(`curl|sh`,`wget|sh`)。次を Confirm: `sudo`・到達範囲広いコマンド**。パターンは設定で追加可【推奨】
- **B**: 最小限（`rm -rf /` 等の即死系のみ）
- **C**: その他（自由記述）

[Answer]:

---

### Q5. Git の危険操作（US-4.4/5.2）
- **A**: **通常(commit/branch/add/status/diff/log)=Allow。`push --force`/`push -f`・`reset --hard`・履歴改変(`rebase`,`filter-branch`,`commit --amend`の push 連動)・`clean -fdx` = Confirm/Deny**。リモートpushは既定 Confirm【推奨】
- **B**: 全Git操作 Allow（MVP簡素）
- **C**: その他

[Answer]:

---

### Q6. Web ツール（FR-4.4, US-4.5）
- **A**: **GET のみ（既定）。http/https のみ。攻撃的アクセス（ポートスキャン的連打・内部ネットワーク探索）を抑止。ユーザー指示由来のみ**。本文はサイズ上限で取得【推奨】
- **B**: GET/POST 両対応
- **C**: その他

[Answer]:

---

### Q7. 確認(Confirm)の伝達インタフェース（US-5.2）
ガードレールが「確認が必要」をどう人間に問うか（実装はU5だが境界をここで定義）。
- **A**: **`Confirmer` インタフェース**（`Confirm(ctx, action, reason) (bool, error)`）を U3 で定義し DI。**非対話(非TTY/CI)時は確認不能 → 安全側で Deny 相当に倒す**（フェイルクローズ）【推奨】
- **B**: その他

[Answer]:

---

### Q8. フェイルクローズの具体（US-6.2, SECURITY-15）
- **A**: **判定中のエラー/未知の操作種別/スコープ判定不能 → Allowにしない（Confirm か Deny）。ツール実行失敗は中間状態を残さないよう努め、失敗を明示して停止/再判断**【推奨】
- **B**: その他

[Answer]:

---

### Q9. その他（任意）
ツール名の規約、Webの取得サイズ上限、ターミナルのシェル(`sh -c`)利用可否、追加で塞ぎたい操作など。

[Answer]:
