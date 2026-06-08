# NFR Requirements — U1 Foundation

**Unit**: U1 Foundation（Config, Logging）。規約: TDD + 単体 + PBT(rapid)、依存最小化(SECURITY-10)。

## 適用判定（U1スコープ）
| NFR領域 | 判定 | 内容 |
|---|---|---|
| Scalability | **N/A** | 単一プロセス、設定は起動時1回ロード。並行スケール対象なし |
| Availability / DR | **N/A** | ローカルCLI、永続状態なし(Q6=A)。フェイルオーバー概念なし |
| Performance | 軽微要件 | NFR-P1/P2 参照 |
| Security | 要件あり | NFR-S1〜S3（Functional R4/R5/R6を継承） |
| Reliability | 要件あり | NFR-R1（フェイルクローズ） |
| Maintainability | 要件あり | NFR-M1（TDD/PBT/依存最小化） |
| Usability | 軽微要件 | NFR-U1（分かりやすいエラー） |

## Performance
- **NFR-P1**: 設定ロード（ファイル読込+マージ+検証）は体感即時。目安 < 数十ms（実測はBuild and Testで確認）。
- **NFR-P2**: ログ出力はMVPでは**同期**で可。ホットパスで過度なアロケーション/文字列連結を避ける。非同期化は将来の最適化（今回要件化しない, Q5=A）。

## Security（Functional Designのルールを継承・再確認）
- **NFR-S1**（SECURITY-05）: 設定入力検証（endpoint URL / maxSteps>0 / workspace 解決）。→ business-rules R4。
- **NFR-S2**（SECURITY-09）: 既定にsecretを持たない、エラーは内部情報を露出しない。→ R5, R4。
- **NFR-S3**（SECURITY-03）: ログの機微情報マスキング（トークン/認証/プロンプト本文）。→ R6。

## Reliability
- **NFR-R1**（SECURITY-15）: 設定不正は**フェイルクローズ**（起動中止・非0終了）。ログファイルオープン失敗のみ非致命でstderrフォールバック。→ F1。

## Maintainability
- **NFR-M1**: TDD（test-first）。PBT(rapid)を R1/R4(URL)/R6/R8 に適用。依存は最小（標準ライブラリ優先, SECURITY-10）。`go.sum` 固定、`govulncheck`（Build and Test）。

## Usability
- **NFR-U1**: 検証エラー文言は「どの設定キーが・なぜ不正か・どう直すか」を平易に提示（US-6.1の基盤）。

## テスト可能な受け入れ観点
- ロードのウォームパスにファイルI/Oが1回以下（探索2ファイル）であること（P1）。
- マスク後出力に生シークレットが現れないこと（S3, PBT）。
- 不正設定で必ず非0終了し、stdout/stderrに内部パスやスタックを出さないこと（S2, R1）。
