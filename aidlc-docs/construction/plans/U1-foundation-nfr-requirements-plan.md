# NFR Requirements Plan — U1 Foundation（CONSTRUCTION / Part 1 Planning）

**Unit**: U1 Foundation（Config, Logging）
**規約**: TDD + 単体 + PBT（rapid）。SECURITY-10（依存最小化）を重視。

> U1は単一プロセス内のローカル処理であり、**スケーラビリティ/可用性/スループットの論点は実質N/A**（後述）。論点は主に**技術スタック選定**と最低限の性能・信頼性。各 `[Answer]:` を記入。「おまかせ」で推奨採用。

## NFR 早見（U1スコープ）
| NFR領域 | U1での扱い |
|---|---|
| Scalability | N/A（単一プロセス・起動時1回のロード） |
| Availability/DR | N/A（ローカルCLI、状態永続なし Q6=A） |
| Performance | 軽微: 設定ロードは起動時1回。ログはホットパスになりうる→過度なアロケーション回避（後段で計測） |
| Security | R4/R5/R6（検証・secret非保持・マスキング）= Functional Designで定義済み。ここで再掲し充足を確認 |
| Reliability | フェイルクローズ（R4, F1） |
| Maintainability | TDD + PBT、依存最小化 |
| Usability | エラーメッセージの分かりやすさ（R4, US-6.1の前提） |

## 1. 生成プラン（Part 2で実施）
- [ ] `construction/U1-foundation/nfr-requirements/nfr-requirements.md`
- [ ] `construction/U1-foundation/nfr-requirements/tech-stack-decisions.md`

---

## 2. 確認質問（主に技術選定）

### Q1. 構造化ログの実装
- **A**: 標準 `log/slog`（Go 1.21+ 標準、**追加依存なし**、JSON/text・レベル・属性対応）【推奨, SECURITY-10】
- **B**: サードパーティ（zap / zerolog 等、高速だが依存増）
- **C**: 自前実装
- **D**: その他

[Answer]:

---

### Q2. YAML設定パーサ
- **A**: `gopkg.in/yaml.v3`（事実上の標準、安定）【推奨】
- **B**: 別ライブラリ（自由記述）
- **C**: 設定ファイル形式自体を見直す（例: 標準ライブラリだけで済むJSON）→ Functional Q2の再検討

[Answer]:

---

### Q3. CLIフラグ解析（U1のConfigロードに必要な範囲）
Application Design Q9暫定は標準`flag`。確定しますか？
- **A**: 標準 `flag` で確定【推奨, SECURITY-10】
- **B**: `cobra` / `urfave/cli` を採用（自由記述）
- **C**: U5（CLI unit）で最終決定するので保留

[Answer]:

---

### Q4. Goの最低バージョン
- **A**: **Go 1.22+**（`log/slog` 安定、最近の標準機能を利用可）【推奨】
- **B**: 別バージョン指定（自由記述）

[Answer]:

---

### Q5. パフォーマンス目標（U1の最小限）
- **A**: 設定ロードは体感即時（数十ms以内）、ログは同期出力でMVP十分（高スループット最適化は後回し）【推奨】
- **B**: ログは非同期/バッファリングを最初から要件化する
- **C**: その他

[Answer]:

---

### Q6. その他（任意）
依存の方針、ライセンス互換（GPL-3.0と依存ライブラリの互換）で気になる点など。

[Answer]:
