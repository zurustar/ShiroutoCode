# Code Generation Plan — U1 Foundation（CONSTRUCTION / Part 1）

**Unit**: U1 Foundation（Config, Logging）
**規約**: **TDD（test-first: red→green→refactor）** + PBT(rapid)。本plan が Code Generation の単一の真実。
**プロジェクト型**: Greenfield / 単一Goモジュール・単一バイナリ（monolith）。
**ワークスペースルート**: `/Users/oumi/Documents/GitHub/ShiroutoCode`（コードはルート、ドキュメントは aidlc-docs のみ）。

## 担当ストーリー（U1）
- US-2.1（設定: エンドポイント/モデル指定）主
- US-3.3/5.3/6.1 へ設定・ワークスペースルートを提供（本unitでは「提供基盤」まで）
- 横断: 機微情報マスキング（SECURITY-03）

## 依存
- 他unitへの依存なし（U1は土台）。`Frontend`/ツール等は未関与。

## コード配置（確定）
```
/ (workspace root)
├── go.mod / go.sum                 # module: github.com/zurustar/shiroutocode, go 1.22
├── internal/log/
│   ├── log.go                      # Logger, slog構成, maskingHandler, MaskRuleSet
│   └── log_test.go                 # 単体 + PBT(マスク/レベル)
└── internal/config/
    ├── config.go                   # Config, Load, sources, merge, validate
    └── config_test.go              # 単体 + PBT(優先順位/URL検証)
```
※ ドキュメント要約は `aidlc-docs/construction/U1-foundation/code/` に出力（markdownのみ）。

## このバイナリの方針メモ
U1単体では `main` を作らない（CLIはU5）。`go test ./...` が green になることがU1の完了条件（Q3=A）。

---

## 生成ステップ（TDD・順次実行）

### [x] Step 1: プロジェクト構造セットアップ（greenfield）
- `go.mod` 作成（module `github.com/zurustar/shiroutocode`, `go 1.22`）
- 依存追加: `gopkg.in/yaml.v3`, `pgregory.net/rapid`（test）
- `.gitignore` にビルド成果物（`/bin/`, `*.out` 等）追記（必要なら）

### [x] Step 2: Logging — テスト先行（RED）
`internal/log/log_test.go` を先に作成（まだ実装なし＝失敗）:
- マスク: 任意属性で `authorization/token/api_key/secret/password` 値が出力に生で出ない（**PBT/rapid**, R6）
- マスクの冪等性（R6）
- プロンプト本文は既定で要約、`debug` で全文（R6）
- レベルフィルタ: 設定レベル未満は出力されない（**PBT**, R8）
- 全レコードに timestamp/level、`With(correlationID)` が全件に伝播（R7）

### [x] Step 3: Logging — 実装（GREEN→REFACTOR）
`internal/log/log.go`:
- `Logger` インタフェース（Info/Warn/Error/With）＋ slogベース実装
- `maskingHandler`（slog.Handler デコレータ, P1）, `MaskRuleSet`（LC4）
- `New(cfgっぽい引数: level/format/writer)` 、text/json、`io.Writer` 注入可
- Step 2 のテストが green になるまで実装・リファクタ

### [x] Step 4: Config — テスト先行（RED）
`internal/config/config_test.go` を先に作成:
- 優先順位 flag>env>project>home>default（**PBT/rapid**, R1 全単射性）
- 環境変数 `SHIROUTO_*` マッピング（R2）
- YAMLファイル探索/不在は正常/不正YAMLはエラー（R3）
- 検証: model必須・endpoint URL（**PBT**で正/不正URL, R4）・maxSteps>0・workspace解決
- エラー集約 `errors.Join`（複数違反同時提示, P2）
- 既定にsecretを含まない（R5）
- 「未設定」と「ゼロ値」を区別（P3）

### [x] Step 5: Config — 実装（GREEN→REFACTOR）
`internal/config/config.go`:
- `Config` / `GuardrailPolicy` / `ConfigSource` 型（domain-entities準拠）
- `Load(args, env, opts)`：sources読込→段階的上書きマージ(P3)→検証集約(P2)→不変Config
- ソースリーダ（defaults/yaml(home,project)/env/flag）はテスト可能に分離（LC2）
- フェイルクローズ（P4）

### [x] Step 6: コード要約ドキュメント
`aidlc-docs/construction/U1-foundation/code/` に markdown:
- `code-summary.md`（生成物一覧・設計対応・拡張コンプライアンス）
- `test-summary.md`（テスト観点とPBTプロパティの対応表）

### [x] Step 7: ローカル検証（TDDの締め）
- `go build ./...` と `go test ./...`（rapid含む）を実行し **green** を確認
- 結果を `test-summary.md` に追記（Build and Test ステージで再実行）
- （注: Goがローカル未導入の場合は、その旨を報告し Build and Test 段で実行に回す）

### デプロイ成果物 / マイグレーション / API / Frontend
- **N/A**（U1にDB・API・UIなし。バイナリ配布はU5/Build and Test）

---

## ストーリートレーサビリティ
| Story | 実装ステップ | 完了マーク条件 |
|---|---|---|
| US-2.1（設定） | Step 4,5 | Config Load+検証 green |
| 横断マスキング | Step 2,3 | マスクPBT green |
| US-3.3/5.3/6.1 基盤 | Step 5 | maxSteps/workspace/endpoint を Config が提供 |

## スコープ概算
- 7ステップ、生成ファイル: `go.mod` + 4 Goファイル（2実装/2テスト） + 2要約md。新規依存2（yaml.v3, rapid[test]）。
