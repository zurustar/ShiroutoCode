# Business Rules — U1 Foundation

> 各ルールは **テスト可能な表明** として記述（TDD: 先に失敗するテストを書く）。`[PBT]` 印は Property-Based Test（rapid）候補。

## 設定マージ・優先順位

### R1. 優先順位（US-2.1, FR-6）
実効値は次の優先順で決定: **Flag > Env > ProjectFile > HomeFile > Default**。
- 表明: あるキーが複数ソースに存在するとき、最も優先度の高いソースの値が採用される。
- 表明: どのソースにも無い任意項目は既定値、必須項目は欠如エラー（R4）。
- `[PBT]` 任意のソース集合・任意のキー集合に対し、採用値は「そのキーを持つ最高優先ソースの値」に一致する（優先順位の全単射性）。

### R2. 環境変数マッピング
prefix `SHIROUTO_` + 大文字スネーク。例: `SHIROUTO_ENDPOINT`, `SHIROUTO_MODEL`, `SHIROUTO_MAX_STEPS`, `SHIROUTO_WORKSPACE`, `SHIROUTO_LOG_LEVEL`。
- 表明: `SHIROUTO_MODEL=foo` のみ与えた場合、他は既定で `Model=foo`。

### R3. 設定ファイル探索（Q2=A）
形式 **YAML**。探索: ProjectFile=`<cwd>/.shiroutocode.yaml`、HomeFile=`~/.config/shiroutocode/config.yaml`。
- 表明: 両方存在時、ProjectFile が HomeFile より優先（R1）。
- 表明: ファイル不在は**エラーではない**（既定/他ソースで補完）。
- 表明: ファイルは存在するが不正YAML → 検証エラー（R4、フェイルクローズ）。

## 検証（SECURITY-05 / 09）

### R4. 検証失敗時は即エラー終了（Q4=A, フェイルクローズ）
起動時に実効Configを検証し、不正なら**起動を中止**し終了コード≠0。
- `model` 必須: 未設定なら「modelが未設定です。--model か SHIROUTO_MODEL か設定ファイルで指定してください」。
- `endpoint` はURL形式（scheme http/https + host）であること。`[PBT]` 不正URL集合は常に拒否、正当URL集合は常に受理。
- `maxSteps > 0`。0以下は拒否。
- `workspace` は存在するディレクトリへ解決できること（パスは絶対化・正規化）。
- 表明: 複数の不正があるとき、可能な限りまとめて提示する（最初の1件だけでない）。
- 表明（SECURITY-09）: エラーメッセージは内部パス/スタックトレース/値の生ダンプを露出しない（どの設定キーが・なぜ不正か、だけ）。

### R5. デフォルト認証情報を持たない（SECURITY-09）
- 表明: いかなる既定値にもトークン/パスワード/APIキーを含めない（LM Studioはローカルで通常認証不要）。

## マスキング（SECURITY-03）

### R6. 機微情報マスク（Q5=A）
ログ出力前に LogRecord の Message/Fields に MaskRule を適用。
- キー名一致（大小無視）: `authorization`, `token`, `api_key`, `apikey`, `secret`, `password` → 値を `***` に。
- プロンプト本文（LLM送受信テキスト）は既定でマスク/省略（例: `<prompt:NNN chars>`）。`LogLevel=debug` のときのみ全文許可。
- `[PBT]` 任意のキー/値入力に対し、マスク対象キーの値が出力に**生のまま現れない**（マスク後出力に元シークレットが部分文字列として残らない）。
- 表明: マスクは冪等（二重適用しても結果不変）。

## ログ出力（NFR-5）

### R7. 出力先・形式（Q6=A）
- 既定: **stderr** に人間可読テキスト。
- `LogFormat=json`（`--log-format=json`）: 1行1JSONの構造化出力（Timestamp/Level/Message/CorrelationID/Fields）。
- `LogFile` 指定時: そのファイルへ追記（失敗時はstderrにフォールバックし警告）。
- 表明: 全LogRecordに Timestamp と Level が必ず含まれる。
- 表明: `With(correlationID)` 後のログは全件その相関IDを持つ。

### R8. レベルフィルタ
出力は `LogLevel` 以上のみ（debug<info<warn<error）。
- `[PBT]` 任意レベルのレコード列で、設定レベル未満は出力されず、以上は出力される。
