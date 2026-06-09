# Integration Test Instructions

## Purpose
unit 間の結合（CLI→Agent→Guardrail/LLM/Tools）が協調動作することを確認する。LM Studio 不要（モック使用）。

## 既存の自動結合テスト
| シナリオ | 場所 | 内容 |
|---|---|---|
| CLI → Agent → (fake LLM) | `internal/cli/integration_test.go::TestSingleShotCompletes` | 単発実行で完了→stdout要約・exit0 |
| CLI → Agent → 接続エラー | `…::TestSingleShotConnectionError` | Unreachable→US-6.1案内・exit≠0・内部非露出 |
| Agent → Dispatcher(実Evaluator) → ブロック | `internal/guardrail/dispatcher_test.go::TestDispatchRealEvaluatorScopeDeny` | workspace脱出書込を阻止 |
| Agent ループ（fake LLM/Dispatcher） | `internal/agent/agent_test.go` | ツール→完了、maxStep停止、中断 |

```bash
go test ./internal/cli/ ./internal/agent/ ./internal/guardrail/ -count=1 -race
```

## 手動結合スモーク（実バイナリ、LM Studio不要）
```bash
make build
# 設定不足 → exit 2
echo | ./bin/shiroutocode ; echo "exit=$?"
# モデル設定・LM Studio未起動 → 接続案内 + exit≠0（US-6.1）
SHIROUTO_MODEL=dummy SHIROUTO_WORKSPACE=/tmp ./bin/shiroutocode "hi" ; echo "exit=$?"
```
**実測**: 前者は「設定エラー」exit 2、後者はリトライ後「LM Studio に接続できません…」exit≠0。

## Cleanup
一時ファイルなし（テストは `t.TempDir()`/`httptest` で自動破棄）。
