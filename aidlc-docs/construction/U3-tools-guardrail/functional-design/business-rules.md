# Business Rules — U3 Tools & Guardrail

> テスト可能な表明（TDD）。`[PBT]` は rapid 候補。**安全性の中核**＝過剰ブロック側に倒す（fail-safe）。

## ディスパッチ（単一窓口）

### R1. 全ツール実行は ToolDispatcher 経由（App Q5=A）
- 表明: Tool は Registry 経由でのみ取得され、実行は ToolDispatcher.Dispatch のみが行う（Agentが直接 Tool.Execute を呼ぶ経路は存在しない＝設計・レビューで担保）。
- 表明: Dispatch は必ず Evaluate を呼んでから実行可否を決める。

### R2. 判定→実行の対応（Q2=A）
- `Allow` → そのまま実行。
- `Confirm` → `Confirmer.Confirm` が true のときのみ実行、false なら未実行で「スキップ（拒否）」を返す。
- `Deny` → 実行せず理由を返す（Confirmerにも問わない）。
- `[PBT]` 任意の Decision で、実行が起きるのは (Allow) または (Confirm かつ確認true) のときに限る。

## スコープ限定（US-5.3, SECURITY-11）

### R3. ワークスペース実体パス封じ込め（Q3=A）
- 表明: File 系の対象パスは絶対化・`filepath.Clean`・**シンボリックリンク解決後**の実体が `WorkspaceRoot` 配下であること。
- 表明: 書込/削除がスコープ外 → `Deny`。読取がスコープ外 → `Confirm`。
- 表明: `..` やシンボリックリンクを使った脱出は実体解決により阻止される。
- `[PBT]` `WorkspaceRoot` 配下の任意の相対パスは内部判定で「内」。`../` を十分前置した任意パスは「外」。
- 表明（フェイルクローズ）: パス解決でエラー（不正パス等）→ 内と見なさない（Deny/Confirm）。

## 危険コマンド denylist（US-5.2, Terminal）

### R4. Deny パターン（Q4=A）
次にマッチするコマンドは `Deny`:
- ルート/広域削除: `rm -rf /`, `rm -rf /*`, `rm -rf ~`, `rm` の絶対パスでワークスペース外
- デバイス/低レベル: 書込先 `/dev/...`, `dd of=/dev/...`, `mkfs`
- システム制御: `shutdown`, `reboot`, `halt`, `init 0/6`
- fork爆弾: `:(){ :|:& };:` 等のパターン
- パイプ実行: `curl ... | sh`, `wget ... | sh`(bash/zsh含む)
- 認証情報の外部送信が明白なもの（ベストエフォート）

### R5. Confirm パターン（Q4=A）
- `sudo`、権限昇格、`chmod -R`/`chown -R` の広域、ワークスペース外への `mv`、ネットワーク到達が広いコマンド。
- 表明: denylist/confirmlist は設定（`ExtraDenyPatterns`）で追加可能。
- `[PBT]` 既知の各 Deny パターン文字列（バリエーション: 余分な空白/オプション順）に対し、判定は常に Deny（過剰側）。
- 表明（フェイルクローズ）: パース不能・判定不能なコマンドは Allowにしない（Confirm）。

## Git 危険操作（US-4.4/5.2）

### R6. Git 判定（Q5=A）
- Allow: `status`/`diff`/`log`/`add`/`commit`/`branch`/`checkout`/`switch`/`stash`/`pull(fast-forward)`。
- Confirm: `push`（リモート反映、既定確認）/`merge`/`rebase`。
- Deny もしくは Confirm（ConfirmModeに従う）: `push --force`/`-f`, `reset --hard`, `rebase`(履歴改変), `filter-branch`, `clean -fdx`, `commit --amend`+force push 連動。
- `[PBT]` force/hard/履歴改変フラグを含む git コマンドは Allow にならない。

## Web ツール（FR-4.4, US-4.5）

### R7. Web 制限（Q6=A）
- GET のみ。スキーム http/https のみ（file://, ftp:// 等は Deny）。
- 表明: ユーザー指示由来の取得のみ（エージェントが勝手に大量アクセスしない＝呼び出し回数はエージェントループ制御に従う）。
- 表明: レスポンスはサイズ上限（既定 例 1MiB）で打ち切り（メモリ保護）。
- 表明: 攻撃的（同一ホスト連打/ポート走査的パターン）は抑止対象（ベストエフォート、Deny/Confirm）。

## 確認・フェイルクローズ（US-5.2/6.2, SECURITY-15）

### R8. 確認の伝達と非対話（Q7=A）
- 表明: `Confirm` 判定時、`Confirmer` が対話可能なら諾否を問う。**非対話（Confirmer未提供/非TTY）では確認できない → 実行しない（Deny相当）**。
- `[PBT]` 非対話 Confirmer 下では、Confirm/Deny いずれの判定でも実行は起きない。

### R9. フェイルクローズ（Q8=A）
- 表明: 評価中のエラー・未知の ActionKind・スコープ判定不能 → Allow にしない。
- 表明: ツール実行失敗時は失敗を明示し、危険側に進まない（中間状態を残さない努力: 例 編集は一時書込→rename 等）。

## 自動承認（US-5.1）

### R10. デフォルト自動承認
- 表明: 上記いずれの危険判定にも該当しないワークスペース内の通常操作は `Allow`（ユーザー確認なしで実行）。
- 表明: 実行された操作は結果（ToolResult）として記録され、事後に履歴で確認できる（U5表示）。

## 横断
- 全ツールの引数・パス・コマンドは検証（SECURITY-05）。判定ロジックは専用パッケージ `internal/guardrail` に分離（SECURITY-11）。ログはU1マスキング前提。
