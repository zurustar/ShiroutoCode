# Code Summary — U1 Foundation

**生成日**: 2026-06-08 / **規約**: TDD（test-first）+ PBT(rapid)。**全テスト green 確認済み**（Step 7）。

## 生成ファイル（アプリコード = ワークスペースルート）
| パス | 種別 | 内容 |
|---|---|---|
| `go.mod` / `go.sum` | 新規 | module `github.com/zurustar/shiroutocode`, go 1.22。依存: `gopkg.in/yaml.v3`, `pgregory.net/rapid`(test) |
| `internal/log/log.go` | 新規 | `Logger`(slog), `maskingHandler`(P1デコレータ), `MaskRuleSet`, `maskValue`/`maskAttr` |
| `internal/log/log_test.go` | 新規 | 単体6 + PBT(マスク冪等/レベルフィルタ) |
| `internal/config/config.go` | 新規 | `Config`/`GuardrailPolicy`, `Load`(段階的上書きP3), `validate`(集約P2), env/yaml/flag リーダ |
| `internal/config/config_test.go` | 新規 | 単体7 + PBT(優先順位R1/URL検証R4) |

## 設計対応
| 設計要素 | 実装 |
|---|---|
| business-rules R1 優先順位 | `Load` の overlay 順 (home→proj→env→flag)、`TestPrecedencePBT` |
| R2 環境変数 `SHIROUTO_*` | `readEnv` |
| R3 ファイル探索/不在/不正YAML | `readYAMLFile`（`fs.ErrNotExist`は正常、unmarshal失敗はerror） |
| R4 検証+集約 | `validate` + `errors.Join`、`validateEndpoint` |
| R5 既定にsecretなし | `defaults`、`TestDefaultsHaveNoSecrets` |
| R6 マスキング | `maskValue`/`maskAttr`/`maskingHandler` |
| R7/R8 出力・レベル | `New`(text/json, level), `With` 伝播 |
| P1 マスクデコレータ | `maskingHandler` が基底Handlerをラップ |
| P2 エラー集約 | `errors.Join` |
| P3 段階的上書き | `partial`(ポインタで presence)、`overlay` |
| P4 フェイルクローズ | `Load` はエラー時 `Config{}`+err（呼び出し側が非0終了） |

## 拡張コンプライアンス（U1 Code）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 | ✔ | `validate`/`validateEndpoint`/`SHIROUTO_MAX_STEPS`数値検証 |
| SECURITY-09 | ✔ | 既定secretなし、検証エラーは内部パス/スタック非露出の文言 |
| SECURITY-03 | ✔ | マスキング（生シークレット非出力、PBT検証） |
| SECURITY-15 | ✔ | フェイルクローズ（検証失敗で読み込み中断） |
| PBT-09 (rapid) | ✔ | R1/R4(URL)/R6/R8 をPBT化 |
| SECURITY-10 | 一部 | 依存最小(yaml.v3 + rapid[test])。`go.sum`固定。`govulncheck`はBuild&Test |
| 他SECURITY | N/A | クラウド/認証/データストア前提 |

## 注記
- U1単体では `main` 無し（CLIはU5）。完了条件＝`go test ./...` green（Q3=A）を満たす。
