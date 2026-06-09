# Test Summary — U4 Agent Engine

**実行**: `go test ./... -count=1 -race` → **PASS**（2026-06-10）。gofmt/vet クリーン。U4: 6テスト（PBT 1）。

| テスト | 種別 | ルール |
|---|---|---|
| TestSingleShotCompletes | unit | R2（ツールなし→Completed, OnAssistantText/OnStep） |
| TestToolThenComplete | unit | R1/R5（2ステップ, Dispatch経由, changed集約, OnToolCall） |
| TestMaxStepsTerminationPBT | **PBT** | R3（常時ツール→exactly maxSteps で停止, 無限ループ不可） |
| TestCancelAborts | unit | R4（事前キャンセル→Aborted, Dispatch呼ばない） |
| TestBlockedToolContinues | unit | R6（ツールerr→観測化→継続→Completed） |
| TestEmptyPromptFails | unit | R7 |

## メモ
- fake LLM（スクリプト化 Stream）・fake Dispatcher・記録 Frontend で hermetic にループ検証（実LLM/実ツール不要）。
- 停止性（暴走防止 US-3.3）を PBT で保証。-race クリーン。
- 実環境E2EはU5/Build and Test。
