# NFR Design Plan — U3 Tools & Guardrail（CONSTRUCTION / Part 1）

**Unit**: U3。規約: TDD + PBT(rapid)。安全パターンの具体化が主。

## 1. 生成プラン（Part 2で実施）
- [x] `construction/U3-tools-guardrail/nfr-design/nfr-design-patterns.md`
- [x] `construction/U3-tools-guardrail/nfr-design/logical-components.md`

---

## 2. 確認質問

### Q1. スコープ封じ込めの実装（R3/NFR-S2）
- **A**: **`resolveWithin(root, target)`: 絶対化→Clean→`EvalSymlinks`（存在する祖先まで）→`root` 配下か判定。新規作成パスは「存在する最深の親」を解決して判定**。判定不能はエラー＝非許可【推奨】
- **B**: その他

[Answer]: A（おまかせ）

---

### Q2. denylist ルール表の構造（R4-R7/NFR-M1）
- **A**: **`[]Rule{ Kind, Matcher(正規表現/述語), Decision, Reason }` の順序付きデータ表**。最初にマッチした Deny>Confirm を採用、無ければ Allow。Config の追加パターンは Deny として後段マージ【推奨】
- **B**: switch文の直書き
- **C**: その他

[Answer]: A（おまかせ）

---

### Q3. コマンド実行の終了制御（R/NFR-R2）
- **A**: **`exec.CommandContext` + プロセスグループ（setpgid）でツリーごとkill**。タイムアウト/ctxキャンセルで確実に停止。出力は上限付き `io.LimitedReader` 相当でストリーム【推奨】
- **B**: 単純な `CommandContext` のみ（孫プロセス残存の可能性）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q4. 原子的ファイル書込（R9/F3）
- **A**: **同一ディレクトリに一時ファイル作成→`os.Rename` で原子的置換**。edit は read→一意置換→原子的書込【推奨】
- **B**: 直接上書き（簡素）
- **C**: その他

[Answer]: A（おまかせ）

---

### Q5. ディスパッチャの構造（R1）
- **A**: **`ToolDispatcher` が Registry/Evaluator/Confirmer/Logger を保持する単一型**。Tool は Dispatcher 経由でのみ実行（公開APIを絞る）【推奨】
- **B**: その他

[Answer]: A（おまかせ）

---

### Q6. その他（任意）

[Answer]: 特になし（おまかせ）
