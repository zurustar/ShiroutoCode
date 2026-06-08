# Tech Stack Decisions — U1 Foundation

| # | 決定事項 | 採用 | 理由 |
|---|---|---|---|
| T1 | 言語 | **Go 1.22+**（実効下限 **1.23**: `rapid` が要求） | NFR-1。`log/slog` 安定、static binary。go.mod は `go 1.23` |
| T2 | 構造化ログ | **標準 `log/slog`** | 追加依存なし(SECURITY-10)。JSON/text・レベル・属性・`With`相関ID対応 |
| T3 | 設定ファイル形式/パーサ | **YAML / `gopkg.in/yaml.v3`** | 事実上の標準・安定。Functional Q2=A |
| T4 | CLIフラグ解析 | **標準 `flag`** | 依存最小化(SECURITY-10)。コマンド面が小さいMVPに十分。U5でサブコマンド増なら再検討 |
| T5 | PBTフレームワーク | **`pgregory.net/rapid`** | 拡張PBT-09(Go→rapid)。R1/R4(URL)/R6/R8に適用 |
| T6 | テスト | 標準 `testing` + rapid | TDD。table-driven + property tests |
| T7 | 脆弱性スキャン | `govulncheck` | SECURITY-10（Build and Testで実行） |

## 依存方針（SECURITY-10 / GPL-3.0互換）
- U1で増える外部依存は **`gopkg.in/yaml.v3`（MIT/Apache系, GPL-3.0互換）** と **`pgregory.net/rapid`（MPL/MIT系, テスト専用）** のみ。本体ロジックの依存はほぼ標準ライブラリ。
- `go.mod`/`go.sum` でバージョン固定。`latest` タグ運用はしない。
- すべての依存ライセンスが GPL-3.0 と両立することを Build and Test で確認。

## 後続unitへの申し送り（参考）
- **U5 入力UI**: 利用者決定により **`charmbracelet/bubbletea`（+ bubbles, lipgloss）** を採用予定（フルTUI）。依存増は許容済み。単発実行はTUIなしのプレーン出力。→ U5 NFR/設計で確定。
- U2/U3 等の追加依存（HTTP/SSEは標準 `net/http` で可能）も同方針で最小化を検討。
