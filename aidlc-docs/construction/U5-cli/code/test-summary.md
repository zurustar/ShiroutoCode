# Test Summary — U5 CLI Frontend

**実行**: `go test ./... -count=1 -race` → **PASS**（2026-06-10）。gofmt/vet クリーン。`go build` で単一バイナリ生成。

## テスト一覧（internal/cli）
| テスト | 種別 | 観点 |
|---|---|---|
| TestParseYesPBT | **PBT** | y/yes のみ true（R3） |
| TestPromptConfirmer | unit | y/yes/n/空/その他 と確認文表示 |
| TestPlainFrontend | unit | text/toolCall/result/error/step の区別出力（US-1.2） |
| TestExtractPrompt | unit | フラグと位置引数の分離 |
| TestRunNoPromptNonTTY | unit | 非TTY＋指示なし → usage exit2（R1） |
| TestRunMissingModel | unit | 必須設定欠如 → 設定エラー exit2 |
| TestSingleShotCompletes | integration | fake LLM 完了 → stdout要約・exit0 |
| TestSingleShotConnectionError | integration | Unreachable → 案内文・exit≠0・内部非露出（US-6.1/SECURITY-09） |
| TestTUIAppendsAssistantAndStep | unit | TUI Update: text/step 追記 |
| TestTUIConfirmFlow | unit | confirmReq→y→reply true・状態復帰 |
| TestTUIConfirmDeny | unit | confirmReq→n→reply false |
| TestTUIDoneSummary | unit | done→running解除・要約表示 |
| TestTUICtrlCQuitsWhenIdle | unit | idle時 Ctrl+C→Quit（US-1.3） |

## E2E（実バイナリ smoke）
- 設定不足 → exit2＋「設定エラー」。
- LM Studio未起動 → リトライ後「接続できません」exit≠0（US-6.1 実証）。
- 完全な実LLM対話E2E（マルチファイル編集 US-3.1）は LM Studio 起動環境で手動（Build and Test）。

## メモ
- fake LLM/Dispatcher と Update関数直接呼び出しで hermetic にテスト（実LLM/実端末不要）。-race クリーン。
