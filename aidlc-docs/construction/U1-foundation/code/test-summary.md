# Test Summary — U1 Foundation

**実行**: `go test ./... -count=1` → **PASS**（2026-06-08, Go 1.25.7 toolchain）。`gofmt -l` クリーン、`go vet ./...` クリーン。

## テスト一覧と設計対応
### internal/log（6テスト）
| テスト | 種別 | 検証ルール |
|---|---|---|
| TestMaskingSecretKeys | unit | R6（authorization/token/api_key/secret/password→`***`、生値非出力） |
| TestMaskingIdempotent | **PBT** | R6（マスク冪等性） |
| TestPromptSummarizedUnlessDebug | unit | R6（プロンプト本文: info要約 / debug全文） |
| TestLevelFilter | **PBT** | R8（レベル閾値で出力可否） |
| TestTimestampLevelAndCorrelation | unit | R7（time/level必須、`With`相関ID伝播） |
| TestTextFormatMasks | unit | R7（text形式でもマスク） |

### internal/config（9テスト）
| テスト | 種別 | 検証ルール |
|---|---|---|
| TestDefaults | unit | 既定値（endpoint/maxSteps/log/guardrail） |
| TestEnvMapping | unit | R2（`SHIROUTO_*`） |
| TestYAMLProjectOverridesHome | unit | R1/R3（project>home、home値の残存） |
| TestInvalidYAMLIsError | unit | R3（不正YAML→error） |
| TestValidationAggregatesErrors | unit | R4/P2（model/endpoint/maxSteps/workspace を集約提示） |
| TestZeroValueDistinctFromUnset | unit | R4/P3（maxSteps:0 を拒否＝未設定と区別） |
| TestDefaultsHaveNoSecrets | unit | R5 |
| TestPrecedencePBT | **PBT** | R1（flag>env>project>home>default 全単射） |
| TestEndpointURLValidationPBT | **PBT** | R4（http/https+host受理、その他拒否） |

## PBTカバレッジ（PBT-09 / rapid）
R1（優先順位）・R4（URL検証）・R6（マスク冪等）・R8（レベルフィルタ）= 計4プロパティ。

## 技術メモ
- `go.mod` の go ディレクティブは **1.25**（利用者決定2026-06-08で開発機ツールチェーン1.25.7に一致）。依存`rapid`の下限1.23も内包。ビルド/`go install`に Go1.25+ を要求する点はトレードオフとして許容。
- これらのテストは Build and Test ステージで再実行・CI化される。
