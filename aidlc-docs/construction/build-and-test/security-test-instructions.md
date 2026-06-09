# Security Test Instructions

> Security Baseline 拡張（Blocking）の検証。多くはローカルCLIにつき N/A、関連項目を自動/手動で確認。

## 依存・サプライチェーン（SECURITY-10）
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...        # 脆弱性スキャン（CIにも組込み）
go mod verify            # go.sum 整合
```
- 依存はロックファイルで固定、`latest` タグ運用しない。

## ガードレール（SECURITY-11/05/15）— 自動テストで担保
- スコープ封じ込め（symlink/`../`脱出阻止）: `internal/guardrail` PBT。
- 危険コマンド/Git/Web の denylist: `internal/guardrail` PBT/unit。
- 非対話フェイルクローズ: `dispatcher_test`。
```bash
go test ./internal/guardrail/ ./internal/tools/ -count=1 -race
```

## 機微情報マスキング（SECURITY-03）
- ログにトークン/プロンプト本文が生で出ない: `internal/log` PBT、`internal/llm` UserMessage非漏洩 PBT。

## エラー時の情報露出（SECURITY-09）
- 接続エラー文言が内部詳細を含まない: `internal/cli::TestSingleShotConnectionError`。

## 手動確認（実バイナリ）
- ワークスペース外への書込/削除依頼 → Deny。
- `rm -rf /` 等の依頼 → Deny。`sudo` 等 → 確認要求。
- 非TTY/CIでの危険操作 → 実行されない。

## ライセンス互換（GPL-3.0）
- 追加依存（yaml.v3, rapid, charmbracelet系, x/term）が GPL-3.0 と両立することを確認（MIT/BSD/Apache系）。
