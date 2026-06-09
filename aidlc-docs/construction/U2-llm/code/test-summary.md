# Test Summary — U2 LLM Connectivity

**実行**: `go test ./... -count=1 -race` → **PASS**（2026-06-09）。`gofmt -l` クリーン、`go vet` クリーン。U2: 17テスト（PBT 4）。

## テスト一覧（internal/llm）
| テスト | 種別 | ルール |
|---|---|---|
| TestClassifyHTTPStatuses | unit | R6（404/400→ModelNotFound, 4xx, 5xx retryable） |
| TestClassifyNetErrors | unit | R6（timeout/deadline/refused） |
| TestUserMessageNoLeakPBT | **PBT** | R6/SECURITY-09（UserMessageに本文非漏洩） |
| TestJSONToolRoundTripPBT | **PBT** | R3（tool/final 往復） |
| TestJSONToolToleratesFencesAndProse | unit | R3（コードフェンス/前置き耐性） |
| TestJSONToolUndecodableIsError | unit | R3（不能→Decodeエラー） |
| TestStreamTextReconstructionPBT | **PBT** | R4（text分割→連結保存則） |
| TestToolCallAssemblyPBT | **PBT** | R5（args断片結合） |
| TestStreamCommentsBlanksAndDone | unit | R4（コメント/空行/[DONE]/finish） |
| TestStreamBadJSONIsBadStream | unit | R4（不正data→BadStream） |
| TestCompleteStreamsText | unit | US-2.1/2.2（送信＋逐次text） |
| TestRequestToolsOnlyInFunctionMode | unit | R1（tools送信条件・param省略・json系統prompt注入） |
| TestRetryOn5xxThenSuccess | unit | R7/P2（5xxリトライ→成功、回数） |
| TestNoRetryOn4xx | unit | R7（4xx即失敗・単発） |
| TestAutoFallbackToJSON | unit | R2/P5（function→json 1回フォールバック） |
| TestUnreachableError | unit | US-6.1（接続不可→Unreachable） |
| TestContextCancelAborts | unit | NFR-R2（ctxキャンセルでハングせず終了） |

## PBTカバレッジ（PBT-09 / rapid）
R3（JSON往復）・R4（text保存則）・R5（tool_call結合）・R6（非漏洩）= 4プロパティ。

## メモ
- すべて `net/http/httptest` でモック。-race でデータ競合なし（streamImplのgoroutine readLoopとRecvのchannel/ctx連携を検証）。
- 実 LM Studio 疎通は Build and Test / U5 E2E で実施。
