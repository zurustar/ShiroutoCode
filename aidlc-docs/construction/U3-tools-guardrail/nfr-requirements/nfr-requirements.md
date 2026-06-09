# NFR Requirements — U3 Tools & Guardrail

**Unit**: U3。規約: TDD + PBT(rapid)、依存最小化。Security が支配的NFR。

## 適用判定
| 領域 | 判定 | 内容 |
|---|---|---|
| Scalability | N/A | 単一プロセス・逐次ツール実行 |
| Availability/DR | N/A | ローカル |
| Security | **支配的** | NFR-S1〜S5（Functional R1-R10 を継承・網羅確認） |
| Performance | 軽微 | NFR-P1 |
| Reliability | 要件あり | NFR-R1（フェイルクローズ）、NFR-R2（中断・タイムアウト） |
| Maintainability | 要件あり | NFR-M1（TDD/PBT/データ駆動denylist/依存最小） |
| Usability | 軽微 | NFR-U1（Deny/Confirm理由の明確さ） |

## Security（中核, SECURITY-05/11/13/15）
- **NFR-S1 単一インターセプタ**: 全ツール実行は ToolDispatcher を通る（R1）。バイパス経路を持たない。
- **NFR-S2 スコープ封じ込め**: workspace 実体パス封じ込め（R3、symlink解決）。外部書込/削除は Deny。
- **NFR-S3 危険操作ブロック**: コマンド/Git/Web の denylist・confirmlist（R4-R7）。過剰側に倒す（fail-safe）。
- **NFR-S4 非対話フェイルクローズ**: 確認不能時は実行しない（R8）。
- **NFR-S5 入力検証**: ツール引数/パス/コマンド/URL を実行前に検証（SECURITY-05/13）。判定ロジックは `internal/guardrail` に分離（SECURITY-11）。

## Performance
- **NFR-P1**: ガードレール判定は O(ルール数) の軽量処理（毎実行前に評価）。ボトルネックは外部I/O。MVPで追加最適化なし。

## Reliability
- **NFR-R1**（SECURITY-15）: 判定不能/未知Kind/解決エラー → Allowにしない。編集は原子的書込（一時→rename）。
- **NFR-R2**（NFR-3）: コマンドは ctx 連動タイムアウト（既定 120s, 設定可）。ctxキャンセルでプロセス終了。出力はサイズ上限で打ち切り。Web も同上限・リダイレクト制限。

## Maintainability
- **NFR-M1**: TDD。PBTを R2/R3/R4/R6/R8 に適用。denylist はデータ駆動表＋Config拡張。依存は標準ライブラリ（`os/exec`/`net/http`/`path/filepath`）＋ `git` CLI。

## Usability
- **NFR-U1**: Deny/Confirm には人間可読の理由を付す（U5表示・US-5.2）。

## テスト可能な受け入れ観点
- ツール実行が ToolDispatcher 以外から到達しない（設計・レビュー＋API可視性）。
- symlink/`../` による workspace 脱出が阻止される（PBT）。
- denylist 各パターンが過剰側に判定される（PBT）。
- コマンドタイムアウト/ctxキャンセルでプロセスが終了する。
- 非対話Confirmerで Confirm/Deny とも未実行（PBT）。
