# NFR Design Patterns — U1 Foundation

> NFR要件を実装パターンへ落とし込む（技術: Go 1.22+ / log/slog / yaml.v3 / std flag）。TDDで各パターンに失敗テストを先行。

## P1. マスキング = slog.Handler デコレータ（Q1=A, NFR-S3 / R6）
基底 `slog.Handler`（JSON/text）をラップする `maskingHandler` を実装。`Handle` で `Record` の全属性を走査し、`MaskRule` 適用後に基底へ委譲。
```text
maskingHandler{ inner slog.Handler, rules []MaskRule }
  Handle(ctx, rec):
    rec' = rec.Clone()
    rec'.Attrs( each attr -> applyMask(attr, rules) )   # キー名一致 or プロンプト本文 -> ***/要約
    return inner.Handle(ctx, rec')
  WithAttrs/WithGroup: 同様にラップして伝播（事前付与属性もマスク対象に）
```
- **利点**: 呼び出し側はマスク非意識、抜け漏れにくい。`With(correlationID)` で付与した属性も漏れなく通過。
- **テスト(PBT)**: 任意の属性集合で、マスク対象キーの値が出力に生で現れない（R6, 冪等）。

## P2. 検証エラー集約（Q2=A, R4 / NFR-U1）
各検証関数は `error`（または nil）を返し、`errors.Join(errs...)` で集約。先頭で打ち切らない。
```text
validate(cfg):
    var errs []error
    errs = append(errs, checkModel(cfg), checkEndpointURL(cfg), checkMaxSteps(cfg), checkWorkspace(cfg)) # nilはJoinで無視
    return errors.Join(errs...)   # 全違反を1つのerrへ
```
- ユーザー向け表示は各違反を箇条書き。内部パス/スタックは含めない（SECURITY-09）。
- **テスト**: 複数不正入力 → 返るerrに各違反メッセージが含まれる。

## P3. 段階的上書きマージ（Q3=A, R1）
"設定済みか"を区別するため、各ソースは**部分設定**（任意フィールドが「未設定」を表現できる形）として読み、defaultへ順に重ねる。
```text
eff = defaults()
for src in [HomeFile, ProjectFile, Env, Flags]:   # 低→高優先で上書き
    overlay(eff, src)   # srcで"設定された"フィールドのみ上書き、由来(ConfigSource)を記録
```
- 「未設定」と「ゼロ値（例 maxSteps=0）」を区別する（�值を誤って既定上書きしない）。実装は per-field の presence フラグ or ポインタ等（実装詳細はCode Generation）。
- **テスト(PBT)**: 任意のソース×キー集合で採用値=最高優先ソースの値（R1全単射）。

## P4. フェイルクローズ（NFR-R1 / R4 / SECURITY-15）
- `config.Load` は検証失敗時にエラーを返し、呼び出し側（CLI/main）は非0終了。
- ログファイルオープン失敗のみ非致命: stderrへフォールバックし warn ログ。
- パニックは想定しないが、`main` でトップレベル recover を置き一般化メッセージ化（内部情報非露出）。

## P5. 性能（NFR-P1/P2）
- ロードは起動時1回。ファイル探索は最大2（home/project）。
- ログはMVPでは同期。ホットパスで `slog` の遅延評価（`slog.Attr`）を活用し、debug無効時に重い文字列化を避ける。

## 適用しないパターン（明示）
- リトライ/サーキットブレーカ/バックオフ: U1にネットワーク・外部I/Oの不安定要素が無いため **N/A**（U2 LLMで扱う）。
- キャッシュ/キュー: **N/A**。
