# Business Logic Model — U1 Foundation

> 技術非依存のフロー記述。TDD前提（各フローに対応するテストを先に書く）。

## F1. Config ロードフロー（`config.Load`）
入力: コマンドライン引数、環境変数、（探索した）設定ファイル群。出力: 検証済み `Config` または エラー。

```text
1. 既定値で Config を初期化（ConfigSource=Default）          # R5: secretなし
2. HomeFile (~/.config/shiroutocode/config.yaml) 読込・存在すればマージ   # R3
3. ProjectFile (<cwd>/.shiroutocode.yaml) 読込・存在すればマージ（上書き）  # R1,R3
4. 環境変数 (SHIROUTO_*) をマージ（上書き）                  # R1,R2
5. フラグをマージ（最優先で上書き）                          # R1
6. workspace を絶対化・正規化、存在ディレクトリへ解決         # R4
7. 検証（model必須 / endpoint URL / maxSteps>0 / workspace）  # R4
     ├─ 不正あり → 全不正を収集し一般化メッセージで Error 返却（フェイルクローズ, 非0終了）
     └─ OK       → 不変 Config を返却
```
- 異常系: 不正YAML（R3）、I/Oエラー（権限等）→ 検証エラーに合流。原因キーのみ提示、内部詳細は露出しない（R4/SECURITY-09）。
- ファイル不在は正常系（スキップ）。

## F2. マスキング適用フロー（Logger出力時）
```text
LogRecord 受領
  → Message と Fields の各要素に MaskRule を適用            # R6
       ├─ キー名がマスク対象 → 値を *** に
       ├─ プロンプト本文フィールド → debug未満なら要約表現に
       → マスク後の LogRecord を生成（冪等）
  → LogLevel フィルタ（設定レベル未満は破棄）               # R8
  → LogFormat に従いシリアライズ（text/json）               # R7
  → 出力先へ書き込み（stderr / file、file失敗時stderrへフォールバック+警告）
```

## F3. 相関ID付与
```text
セッション/タスク開始時に CorrelationID を採番（例: 短いランダムID）
  → logger.With(correlationID) で派生ロガーを生成
  → 以降のログは全て同一IDを保持                            # R7表明
```

## エラーハンドリング方針（NFR-4 / SECURITY-15）
- `Load` のエラーは呼び出し側（CLI/main）で捕捉し、人間可読メッセージ + 非0終了。
- ログファイルオープン失敗は致命としない（stderrフォールバック）。それ以外のConfig不正は致命（起動中止）。
- いずれもフェイルクローズ: 不確実な状態で先に進まない。

## テスト観点サマリ（TDD: 先に書く）
| 観点 | 種別 | 対応ルール |
|---|---|---|
| 優先順位の全組み合わせ | PBT | R1 |
| 環境変数マッピング | unit | R2 |
| ファイル探索/不在/不正YAML | unit | R3 |
| 必須欠如・URL不正・maxSteps≤0・workspace不正 | unit + PBT(URL) | R4 |
| 既定にsecretを含まない | unit | R5 |
| マスク（生シークレット非出力・冪等） | PBT | R6 |
| 出力先/形式/相関ID | unit | R7 |
| レベルフィルタ | PBT | R8 |

## 拡張コンプライアンス（U1 Functional Design）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 | 反映 | R4 入力検証（URL/数値/パス） |
| SECURITY-09 | 反映 | R4一般化メッセージ, R5 既定secretなし |
| SECURITY-03 | 反映 | R6 マスキング |
| SECURITY-15 | 反映 | F1/F3 フェイルクローズ |
| PBT-09 (rapid) | 反映 | R1/R4(URL)/R6/R8 をPBT対象に指定 |
| SECURITY-10/11/13 等 | N/A(本unit) | サプライチェーンはBuild段階、ガードレール判定はU3 |
