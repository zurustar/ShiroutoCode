# Test Summary — U3 Tools & Guardrail

**実行**: `go test ./... -count=1 -race` → **PASS**（2026-06-10）。`gofmt -l` クリーン、`go vet` クリーン。U3: 23テスト（PBT 4）。

## テスト一覧
### internal/guardrail
| テスト | 種別 | ルール |
|---|---|---|
| TestScopeContainmentPBT | **PBT** | R3（内rel=内, `../`脱出=外） |
| TestScopeSymlinkEscape | unit | R3（symlink脱出阻止） |
| TestScopeAbsoluteInsideAndOutside | unit | R3 |
| TestCommandDenylistPBT | **PBT** | R4（危険コマンド変種→Deny） |
| TestCommandConfirmAndAllow | unit | R5/R9/R10（sudo→Confirm, 通常→Allow, 空→Confirm） |
| TestGitDestructivePBT | **PBT** | R6（force/hard/履歴改変→非Allow） |
| TestGitNormalAllow | unit | R10（status/add/commit/diff/log→Allow） |
| TestWebScheme | unit | R7（https→Allow, file/ftp→Deny） |
| TestExtraDenyPatterns | unit | Config追加denylist |
| TestUnknownKindConfirms | unit | R9 |
| TestDispatchExecutionInvariantPBT | **PBT** | R2/R8（Allow/Confirm+yesのみ実行） |
| TestDispatchConfirmerErrorFailsClosed | unit | R8/R9（confirmerエラー→未実行） |
| TestDispatchDenyReturnsBlocked | unit | R2 |
| TestDispatchRealEvaluatorScopeDeny | integration | R1+R3（脱出書込を dispatcher が阻止） |

### internal/tools
| テスト | 種別 | ストーリー |
|---|---|---|
| TestFileCreateReadEditDelete | unit | US-4.1/4.2（read/create/edit/delete, 原子的） |
| TestFileEditNonUniqueFails | unit | F3（一意でないedit→エラー） |
| TestTerminalEcho | unit | US-4.3 |
| TestTerminalTimeoutKills | unit | NFR-R2（タイムアウトでgroup kill, 即時） |
| TestTerminalNonZeroExit | unit | US-4.3（非0 exit を結果に） |
| TestGitStatus | unit | US-4.4（git不在時はSkip） |
| TestWebFetchGET | unit | US-4.5 |
| TestWebFetchRejectsNonHTTP | unit | R7 |
| TestWebFetchSizeCap | unit | サイズ上限truncate |

## PBTカバレッジ（PBT-09 / rapid）
R3（スコープ）・R4（denylist）・R6（git）・R2/R8（dispatch実行条件）= 4プロパティ。

## メモ
- ファイル/コマンド/HTTPは `t.TempDir()`/`echo`/`httptest`/`git init` で hermetic に検証。-race クリーン（ターミナルのgoroutine/Cancel経路含む）。
- 実環境E2EはU5/Build and Test。
