# NFR Design Patterns — U3 Tools & Guardrail

> 安全パターンの実装方針。技術: `os/exec`/`net/http`/`path/filepath`。TDDで先行テスト。

## P1. スコープ封じ込め（Q1=A / R3/NFR-S2）
```text
resolveWithin(root, target) (string, error):
  abs = filepath.Abs(target); Clean
  real = EvalSymlinks(abs) if exists
         else: 存在する最深の親を EvalSymlinks し、残り要素を Join（新規作成対応）
  rootReal = EvalSymlinks(root)
  return real, (real==rootReal or strings.HasPrefix(real, rootReal+separator))
  解決エラー → (,false,err)：非許可（フェイルクローズ）
```
- **テスト(PBT)**: root配下の任意相対=内、`../`前置=外、symlink→外部=外。

## P2. denylist ルール表（Q2=A / R4-R7/NFR-M1）
```text
type Rule struct { Kind ActionKind; Match func(Action) bool; Decision Decision; Reason string }
defaultRules() []Rule  // Deny系→Confirm系の順で評価
Evaluate(action):
  scope-check(P1) → 外部書込/削除=Deny, 外部読取=Confirm
  for r in rules where r.Kind==action.Kind: if r.Match(action) return r.Decision,r.Reason
  config ExtraDenyPatterns にマッチ → Deny
  未知Kind/解決不能 → Confirm（フェイルクローズ）
  else Allow
```
- 最初にマッチした Deny が最優先。Matcher は正規表現/述語。空白・オプション順ゆらぎに耐えるよう正規化してマッチ。
- **テスト(PBT)**: 既知Denyパターンの変種は常にDeny。

## P3. コマンド終了制御（Q3=A / NFR-R2）
```text
exec.CommandContext(ctx, ...) で起動
SysProcAttr{Setpgid:true}（Unix）→ タイムアウト/ctxキャンセル時に -pgid へ SIGKILL（ツリー全滅）
stdout/stderr を pipe → goroutine でストリーム（ToolResult.Stream）、合計バイトを上限で打ち切り
終了コードを ExitCode に
```
- **テスト**: sleep するコマンドを短タイムアウトで起動→停止確認（CIで時間に余裕）。出力上限でtruncate。

## P4. 原子的ファイル書込（Q4=A / R9）
```text
writeAtomic(path, data):
  tmp = CreateTemp(dir(path), ".shirouto-*")
  write+fsync→Close→os.Rename(tmp, path)
  失敗時 tmp を削除（中間状態を残さない）
edit: read→old_string一意検証（0件/複数件はエラー）→置換→writeAtomic
```
- **テスト**: edit一意でない→エラー、書込後内容一致、途中失敗で元ファイル不変。

## P5. 単一ディスパッチャ（Q5=A / R1）
```text
type ToolDispatcher struct { reg *Registry; ev *Evaluator; confirmer Confirmer; log log.Logger }
Dispatch(ctx, call) → toAction→Evaluate→(Confirm時)confirmer→(Allow)reg.Get(name).Execute
Tool.Execute は dispatcher 経由でのみ呼ばれる（パッケージAPIで実行口を絞る）
```
- 非対話: confirmer==nil もしくは ErrNonInteractive を返す実装 → Confirm/Deny は未実行（R8）。

## 適用しないパターン
- サーキットブレーカ/プール/キャッシュ: N/A（逐次ローカル実行）。
