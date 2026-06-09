# Business Logic Model — U3 Tools & Guardrail

> 技術非依存フロー。TDD前提。

## F1. Dispatch フロー（単一インターセプタ, R1/R2/R8/R9）
```text
Dispatch(ctx, call):
  1. tool = Registry.Get(call.Name); 無ければ Deny相当のエラー（フェイルクローズ R9）
  2. action = toAction(call)        # 種別/パス/コマンド/URL 抽出
  3. decision, reason = Evaluator.Evaluate(action)
  4. switch decision:
        Deny    → 実行せず {skipped, reason} を返す
        Confirm → ok,err = Confirmer.Confirm(ctx, action, reason)
                    err/対話不可 → 実行しない（Deny相当, R8）
                    ok==false   → スキップ
                    ok==true    → 実行へ
        Allow   → 実行へ
  5. 実行: tool.Execute(ctx, call.Args) → ToolResult（失敗は明示, R9）
  6. ログ（マスキング前提）に判定・操作・結果を記録
```

## F2. Evaluate フロー（R3〜R7/R9/R10）
```text
Evaluate(action):
  scoped-check（File/Git の Paths）:
     解決失敗 → Confirm/Deny（フェイルクローズ R3/R9）
     書込・削除がworkspace外 → Deny
     読取がworkspace外 → Confirm
  種別別ルール:
     Command → denylist(R4)?Deny : confirmlist(R5)?Confirm : パース不能?Confirm : Allow
     GitOp   → forcepush/hard/履歴改変(R6)?Deny/Confirm : push?Confirm : Allow
     WebFetch→ scheme非http(s)/非GET(R7)?Deny : Allow（サイズ上限は実行側）
     FileRead/Write/Delete → スコープ結果に従う
  未知のKind → Confirm（R9）
  どれにも該当しない通常操作 → Allow（R10）
```

## F3. ファイル編集（Q1=A）
```text
write_file(mode):
  create/overwrite: content をワークスペース内パスへ（一時ファイル→rename で原子的, R9）
  edit: ファイル読込 → old_string を一意検索（複数一致/不一致はエラー）→ new_string 置換 → 原子的書込
  delete: スコープ内のみ（ガードレールが既にDeny判定するが二重チェック）
  Changed にパスを記録
```

## F4. ターミナル実行（FR-4.3, App Q4=B）
```text
run_command: ガードレール通過後、ワークスペースを作業ディレクトリに exec
  stdout/stderr をストリーム（ToolResult.Stream へ逐次） / 終了コードを ExitCode
  ctx キャンセルでプロセス終了（中断, US-1.3連携）
```

## F5. Git / Web
```text
git: 通過後 `git <op> <args>` を workspace で実行、出力と Changed を記録
web_fetch: GET、http(s)のみ、サイズ上限で打ち切り、URLを記録
```

## テスト観点（TDD: 先に書く）
| 観点 | 種別 | ルール |
|---|---|---|
| 実行が起きる条件（Allow / Confirm+true のみ） | PBT | R2 |
| スコープ: 内relは内・`../`脱出は外・symlink脱出は外 | PBT + unit | R3 |
| denylist 各パターン（空白/オプション順ゆらぎ）→ Deny | PBT | R4 |
| confirmlist → Confirm、パース不能 → Confirm | unit | R5/R9 |
| git force/hard/履歴改変 → Allowにならない | PBT | R6 |
| web: 非GET/非http(s) → Deny、サイズ上限 | unit | R7 |
| 非対話Confirmerで Confirm/Deny とも実行されない | PBT | R8 |
| 未知Kind/解決エラー → Allowにならない | unit | R9 |
| 通常のworkspace内操作 → Allowで実行・記録 | unit | R10 |
| edit: old_string一意でないとエラー / 原子的書込 | unit | F3 |

## 実装方針メモ（Code Generation向け）
- パッケージ: `internal/tools`(+ file/terminal/git/web サブ or 単一), `internal/guardrail`。
- Evaluator/スコープ/denylist は純粋関数中心＝PBT容易。Tool.Execute は OS 副作用、`t.TempDir()`/`os/exec`（echo等）でテスト。
- `Confirmer` テスト用に「常にyes/常にno/非対話(err)」のフェイクを用意。
- Terminal はMVPで `sh -c` を使うか引数配列かは Q9未指定→ 安全側: **引数配列を基本**、必要時のみ shell。denylistはコマンド文字列に対して評価（shell利用時の `|`/`;`/`&&` 連結も検出）。

## 拡張コンプライアンス（U3 Functional Design）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 | 反映 | 引数/パス/コマンド/URL 検証 |
| SECURITY-11 | 反映(中核) | guardrail 専用パッケージ・単一インターセプタ |
| SECURITY-13 | 反映 | LLM由来引数を検証してから実行（unsafe実行回避） |
| SECURITY-15 | 反映 | フェイルクローズ（判定不能/非対話→実行しない）、原子的書込 |
| PBT-09 (rapid) | 反映 | R2/R3/R4/R6/R8 をPBT化 |
| SECURITY-03 | 継承(U1) | 実行ログのマスキング |
| SECURITY-10 | 反映 | 依存は標準ライブラリ中心（os/exec, net/http, path/filepath） |
