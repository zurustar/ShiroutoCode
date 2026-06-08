# Business Rules — U2 LLM Connectivity

> テスト可能な表明として記述（TDD）。`[PBT]` は rapid プロパティ候補。

## リクエスト組立

### R1. エンドポイント・モデルは設定由来（US-2.1）
- 表明: Request送信先は `Config.Endpoint` + `/chat/completions`、`model` は `Config.Model`。
- 表明: `Tools` は **functionモードのときのみ** リクエストに含める（jsonモードでは送らず、ツール指示はsystem/userプロンプトに埋め込む）。
- 表明: `Temperature`/`MaxTokens` は nil のとき送信ペイロードに含めない（Q7=A）。

### R2. ツールモード解決（ハイブリッド, Q1=A/Q2=A）
- `ToolMode=function`: 常に function calling（`tools` 付き）。
- `ToolMode=json`: function calling を使わず、**単一JSONオブジェクト出力**を要求するプロンプト規約を適用。
- `ToolMode=auto`（既定）: まず function を試行。**フォールバック条件**＝(a) サーバが `tools` 非対応のHTTPエラー/400、(b) 応答にtool_callsが無くツール選択が期待される場面で本文がJSON規約に従っている、(c) function呼び出しが無効。フォールバック後は `Caps.SupportsFunctionCalling=false` を記録し以降jsonモード。
- `[PBT]` 任意のToolModeとサーバ応答パターンに対し、解決されるモードは決定的（同入力→同モード）。

### R3. JSONフォールバック規約（Q2=A）
- 表明: jsonモードでは system プロンプトで「**次のいずれかのJSONのみを出力**: `{"tool":"<name>","args":{...}}` か `{"final":"<text>"}`」を指示。
- パース: 応答テキストから単一JSONを抽出し、`tool`+`args` なら ToolCall、`final` なら最終テキストへ正規化。
- `[PBT]` 生成した整形済みJSON（tool/args または final）は常に正しく往復パースできる。
- 表明: JSONとして解釈不能な応答は `LLMError{Kind:Decode}`（フェイルクローズ: 勝手に実行しない）。

## SSE ストリーミング（US-2.2, Q4=A）

### R4. SSE パース
- 表明: `data: {json}` 行を逐次パースし、`choices[].delta.content` → `TextDelta`、`delta.tool_calls[]` → `ToolCallDelta`。
- 表明: `data: [DONE]` で `Done`（FinishReason付き）を発行しストリーム終了。
- 表明: 空行/コメント行（`:`始まり）は無視。複数 `data:` 行の連結に対応。
- `[PBT]` 任意のテキストをトークン分割して TextDelta 列として流したとき、連結すると元テキストに一致（ストリーム再構成の保存則）。
- 表明: 不正・途中切断のSSEは `LLMError{Kind:BadStream}`。

### R5. ツール呼び出しの蓄積
- 表明: functionモードの `ToolCallDelta`（id/name/args断片）を index ごとに連結し、`Done` 時点で完全な ToolCall 群へ確定。
- `[PBT]` 分割された args 断片を順に結合すると元の引数JSONに一致。

## エラー分類（US-6.1, Q5=A / SECURITY-09）

### R6. エラー分類とユーザー文言
- 接続不可（connection refused/DNS）→ `Unreachable`: 「LM Studio に接続できません。起動状態と Endpoint(URL/ポート) を確認してください」。
- タイムアウト → `Timeout`: 「応答がタイムアウトしました。モデルのロード状況やネットワークを確認してください」。
- HTTP 4xx → `HTTPStatus`(retryable=false)、404/モデル不明 → `ModelNotFound`: 「モデル '<model>' が見つかりません。モデル名を確認してください」。
- HTTP 5xx → `HTTPStatus`(retryable=true)。
- SSE破損 → `BadStream`。
- 表明（SECURITY-09）: `UserMessage` に内部パス/スタック/生レスポンスを含めない。詳細は `wrapped` でログのみ（マスキング前提, U1）。
- `[PBT]` 任意の `LLMError` で `UserMessage` に既知の機微トークン文字列が含まれない。

## リトライ（Q6=A）

### R7. リトライ可否・回数
- 表明: `Retryable=true`（Unreachable/Timeout/5xx）のみ、設定回数（既定2）まで**指数バックオフ**で再試行。`false`（4xx/ModelNotFound/Decode）は即時失敗。
- 表明: リトライは ctx キャンセルで即中断（NFR-3）。
- 表明: ストリーミング開始後（トークン受信後）の中断は原則リトライしない（重複出力防止）。
- `[PBT]` 任意のエラー種別で、実行されるリトライ回数は `retryable ? configured : 0` に一致（上限内）。

## 横断
- すべての外部呼び出しに明示的エラーハンドリング（NFR-4）。
- リクエスト/レスポンス本文をログする場合は U1 のマスキング前提（プロンプト本文は既定要約）。
