# Domain Entities — U1 Foundation

> 技術非依存のドメインモデル。Go型名は方向性の参考（実装詳細はCode Generation）。

## E1. Config（集約ルート）
利用者の実効設定。複数ソースをマージした結果の不変スナップショット。

| フィールド | 型 | 既定 | 必須 | 説明 |
|---|---|---|---|---|
| `Endpoint` | URL文字列 | `http://localhost:1234/v1` | No | LM Studio OpenAI互換ベースURL |
| `Model` | 文字列 | （なし） | **Yes** | 使用モデル名 |
| `MaxSteps` | 整数 | `25`（暫定） | No | エージェントループ上限（>0） |
| `Workspace` | 絶対パス | カレントディレクトリ | No | 操作スコープのルート |
| `Guardrail` | GuardrailPolicy | 後述既定 | No | ガードレール挙動 |
| `LogLevel` | enum(debug/info/warn/error) | `info` | No | ログレベル |
| `LogFormat` | enum(text/json) | `text` | No | 出力形式 |
| `LogFile` | パス \| 空 | 空(=stderr) | No | ログ出力先 |

- **不変性**: `Load` 完了後は読み取り専用（実行中に変化しない）。
- **由来追跡**（任意・デバッグ用）: 各フィールドがどのソース（flag/env/projectFile/homeFile/default）由来かを保持できると検証に有用。

## E2. GuardrailPolicy（値オブジェクト）
ガードレール挙動の構成。U1では「構造の定義」のみ（判定ロジックはU3）。

| フィールド | 型 | 既定 | 説明 |
|---|---|---|---|
| `ConfirmMode` | enum(prompt/deny) | `prompt` | 危険操作時に確認を求めるか、即拒否か |
| `ExtraDenyPatterns` | 文字列[] | 空 | 利用者追加のdenylistパターン |
| `NonInteractivePolicy` | enum(stop/deny) | `stop` | 非TTY時に確認不能な危険操作の扱い（安全側） |

## E3. ConfigSource（列挙）
`Flag`, `Env`, `ProjectFile`, `HomeFile`, `Default`。優先順位の高い順（business-rules R1）。

## E4. LogRecord（値オブジェクト）
構造化ログ1件。

| フィールド | 型 | 説明 |
|---|---|---|
| `Timestamp` | 時刻(RFC3339) | 発生時刻 |
| `Level` | enum | debug/info/warn/error |
| `Message` | 文字列 | 本文（マスク後） |
| `CorrelationID` | 文字列 | セッション/タスク相関ID |
| `Fields` | key→value | 付加フィールド（マスク後） |

## E5. MaskRule（値オブジェクト）
マスキング規則の1単位（business-rules R5）。`KeyPattern`（キー名一致: token/authorization/api_key/secret 等）または `ValuePattern`（値の正規表現）にマッチした要素を `***` 等へ置換。

## 関係
```text
Config 1 ── has ──> 1 GuardrailPolicy
Config 1 ── derivedFrom ──> * ConfigSource（マージ由来）
Logger  ── applies ──> * MaskRule ──> produces ──> LogRecord
```
（ConfigとLoggerは同一unitだが疎結合。LoggerはConfigのLog*フィールドで構成される。）
