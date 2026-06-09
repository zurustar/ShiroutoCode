# NFR Requirements — U4 Agent Engine

**Unit**: U4。規約: TDD + PBT(rapid)、依存最小化。（おまかせ＝推奨で確定）

## 適用判定
| 領域 | 判定 | 内容 |
|---|---|---|
| Scalability | N/A | 単一セッション逐次 |
| Availability/DR | N/A | メモリ会話（永続化なし） |
| Performance | 軽微 | NFR-P1（ストリーミング転送・追加最適化なし） |
| Reliability | 要件あり | NFR-R1（停止性=max steps/ctx）、NFR-R2（フェイルクローズ） |
| Maintainability | 要件あり | NFR-M1（DI/TDD/PBT、依存は標準のみ） |
| Usability | 軽微 | NFR-U1（進行・結果の可視化を Frontend で） |

## 要件
- **NFR-P1**: LLMテキストを `OnAssistantText` で逐次転送（バッファしない）。U4はI/O待ち中心。
- **NFR-R1 停止性（暴走防止, US-3.3）**: ループは必ず終了する（完了 / maxSteps / ctxキャンセル）。**無限ループ不可能**であることをPBTで保証。
- **NFR-R2 フェイルクローズ（NFR-4）**: LLM/ツール/ストリームのエラーで危険側に進まず Failed/Aborted で停止または観測化。ツールは必ず U3 Dispatcher 経由。
- **NFR-M1**: 依存はインタフェース注入（fake LLM/Dispatcher/Frontend でhermeticテスト）。新規外部依存なし。
- **NFR-U1**: OnStep/OnToolCall/OnToolResult/OnAssistantText で進行を可視化（U5が描画）。

## 技術スタック
| # | 決定 | 採用 |
|---|---|---|
| T1 | 言語/標準 | Go、`context` でキャンセル/タイムアウト伝播 |
| T2 | 依存 | **新規外部依存なし**（U1 log/config, U2 llm, U3 tools/guardrail を内部利用） |
| T3 | テスト | `testing` + `rapid` + fake DI |

## テスト可能な受け入れ観点
- 常時ツール要求でも maxSteps で停止（PBT）。
- ctxキャンセルで速やかに Aborted。
- ツール実行が Dispatcher を必ず経由する（fakeで検証）。
