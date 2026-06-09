# Unit Test Execution

## Run Unit Tests
```bash
go test ./... -count=1           # 全テスト
make test                        # 同上
make test-race                   # データ競合検出つき（推奨）
make cover                       # カバレッジ集計
```

## Results (2026-06-10 実測)
- **Total test functions**: 74（全パッケージ）
- **Status**: **全 PASS / 0 failures**、`-race` クリーン、`gofmt -l` クリーン、`go vet` クリーン。
- **Total coverage**: **72.6%**（statements）

| パッケージ | テスト数 | カバレッジ |
|---|---|---|
| internal/config | 9 | 75.7% |
| internal/log | 6 | 89.5% |
| internal/llm | 17 | 83.0% |
| internal/tools | 9 | 66.8% |
| internal/guardrail | 14 | 75.4% |
| internal/agent | 6 | 66.2% |
| internal/cli | 13 | 59.6% |
| cmd/shiroutocode | 0 | (E2E/手動) |

## Property-Based Tests（PBT-09 / rapid）
- config: 設定優先順位、endpoint URL検証
- log: マスク冪等、レベルフィルタ
- llm: JSON往復、SSEテキスト保存則、tool_call結合、UserMessage非漏洩
- tools/guardrail: スコープ封じ込め、コマンドdenylist、git危険操作、dispatch実行条件
- agent: ループ停止性
- cli: parseYes
```bash
go test ./... -run PBT -count=1 -v   # PBTのみ確認
```

## Fix Failing Tests
1. 出力の `--- FAIL` を確認、`go test ./internal/<pkg>/ -run <Test> -v`
2. rapid 失敗時は出力の反例シードで再現（`-rapid.seed`）
3. 修正→再実行
