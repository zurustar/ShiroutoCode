# ShiroutoCode

**ローカルLLMを頭脳とする、自律型AIコーディングエージェント（CLI）。**

ShiroutoCode は [LM Studio](https://lmstudio.ai/) などで動かすローカルLLMに接続し、自然言語の指示からファイル編集・コマンド実行・Git操作・Web取得を多段階で自律実行してタスクを完遂します。クラウドのAPIキーは不要で、コードと指示はローカルの外に出ません。

> 状態: MVP。コアエンジンとCLIフロントエンドを提供します。VSCode拡張などの別フロントエンドは将来フェーズです。

## 特長

- **ローカル完結** — LM Studio の OpenAI互換APIに接続。コードが外部送信されない。
- **自律エージェントループ** — plan → act → observe を繰り返し、1つの指示で複数ファイルにまたがる変更を完遂。
- **5つのツール** — ファイル読み取り / 書き込み / シェルコマンド実行 / Git操作 / Web取得。
- **セーフティガードレール** — 操作はワークスペース内に限定。破壊的・危険な操作は実行前に確認、もしくは拒否。
- **起動後にモデルを選択** — モデル名を覚えていなくてOK。起動時にLM Studioのモデル一覧から矢印キーで選べ、REPLでは `/model` でいつでも切替可能。
- **2つの実行モード** — 単発実行（`shiroutocode "指示"`）と対話セッション（REPL）。
- **単一バイナリ** — Go製。ランタイム依存なしで配布・実行可能。

## 動作要件

- **LM Studio**（またはOpenAI互換の `/v1/chat/completions` を提供するローカルサーバ）。LM Studio側でモデルをロードし、ローカルサーバを起動しておきます（既定 `http://localhost:1234/v1`）。
- 対応プラットフォーム: **Linux (x86_64)** / **macOS (Apple Silicon)**。
- ソースからビルドする場合のみ **Go 1.25 以降**。

## インストール

### 1. リリースバイナリ（推奨）

[Releases](https://github.com/zurustar/ShiroutoCode/releases) から対象プラットフォームのアーカイブを取得して展開します。

```bash
# 例: Linux x86_64
curl -L -o shiroutocode.tar.gz \
  https://github.com/zurustar/ShiroutoCode/releases/latest/download/shiroutocode_<version>_Linux_x86_64.tar.gz
tar xzf shiroutocode.tar.gz
sudo mv shiroutocode /usr/local/bin/

# 例: macOS (Apple Silicon)
# shiroutocode_<version>_macOS_arm64.tar.gz を同様に展開
```

`checksums.txt` で完全性を検証できます（`sha256sum -c`）。

### 2. ソースから（Go）

```bash
go install github.com/zurustar/shiroutocode/cmd/shiroutocode@latest
```

確認:

```bash
shiroutocode --version
```

## クイックスタート

1. LM Studio でモデルをロードし、ローカルサーバを起動します。
2. ワークスペースにしたいディレクトリへ移動します（既定ではカレントディレクトリがワークスペースになります）。
3. そのまま起動します。**モデル名を指定しなければ、起動時に一覧から選べます。**

```bash
# 対話セッション（モデルは起動後にピッカーで選択）
shiroutocode

# 単発実行（モデル未指定なら端末でピッカーが出る）
shiroutocode "READMEに使い方の節を追加して"

# モデルを直接指定することもできる（スクリプト用途など）
shiroutocode --model "google/gemma-4-12b" "テストを追加して"
```

LM Studio のエンドポイントが既定（`http://localhost:1234/v1`）以外なら `--endpoint` で指定します。

### モデルの選択

正確なモデル名を覚えていなくても、起動時にLM Studioのモデル一覧から選べます。

- **REPL**: 起動すると必ずモデル選択ピッカーが開きます。`↑/↓`（または `j/k`）で移動、`Enter` で決定。セッション中はいつでも `/model` と入力すれば再選択できます。
- **単発実行**: `--model`（または `SHIROUTO_MODEL` / 設定ファイル）が指定されていればそれを使い、未指定かつ端末上であればピッカーが開きます。
- **非対話実行（パイプ/CI等）**: ピッカーを出せないため、`--model` などでの指定が必要です。未指定ならエラーになります。

## 使い方

### 単発実行

```bash
shiroutocode [--model <モデル名>] "<指示>"
```

指示を1回処理し、変更内容と結果を出力して終了します。スクリプトやCIに組み込めます（その場合は `--model` などでモデルを指定してください）。

### 対話セッション（REPL）

端末で引数なしに起動すると対話モードになります。最初にモデル選択ピッカーが開き、選ぶとプロンプト入力に移ります。指示を入力するたびにエージェントが実行し、思考過程・ツール呼び出し・ステップ進捗・結果が**逐次（ストリーミングで）**表示されるので、待っている間も動作状況が分かります。入力は端末ネイティブの行編集なので、日本語入力（IME）もそのまま使えます。

コマンド:

| コマンド | 動作 |
|---|---|
| `/model` | モデルを選び直す |
| `/help` | コマンド一覧を表示 |
| `/exit`（`/quit`） | 終了（`Ctrl+D` でも終了） |

### 確認プロンプト

危険となりうる操作（ワークスペース外への影響、`.git/` への書き込み、危険なコマンド等）は、対話実行時に確認を求めます。**非対話実行（パイプ/CI等）では確認できないため、そうした操作は安全側に倒して停止または拒否されます**。

## 設定

設定は次の優先順位でマージされます（上にあるものが優先）。

1. コマンドラインフラグ
2. 環境変数
3. プロジェクト設定ファイル `./.shiroutocode.yaml`
4. ホーム設定ファイル `~/.config/shiroutocode/config.yaml`
5. 組み込みの既定値

### オプション一覧

| 項目 | フラグ | 環境変数 | 設定ファイルキー | 既定値 |
|---|---|---|---|---|
| LLMエンドポイント | `--endpoint` | `SHIROUTO_ENDPOINT` | `endpoint` | `http://localhost:1234/v1` |
| モデル名 | `--model` | `SHIROUTO_MODEL` | `model` | （未指定なら起動時に選択） |
| ワークスペース | `--workspace` | `SHIROUTO_WORKSPACE` | `workspace` | カレントディレクトリ |
| 最大ステップ数 | `--max-steps` | `SHIROUTO_MAX_STEPS` | `maxSteps` | `25` |
| ツール呼び出しモード | `--tool-mode` | `SHIROUTO_TOOL_MODE` | `toolMode` | `auto` |
| ログレベル | `--log-level` | `SHIROUTO_LOG_LEVEL` | `logLevel` | `info` |
| ログ形式 | `--log-format` | `SHIROUTO_LOG_FORMAT` | `logFormat` | `text` |
| ログ出力先ファイル | `--log-file` | `SHIROUTO_LOG_FILE` | `logFile` | （標準エラー） |
| 確認モード | — | — | `confirmMode` | `prompt` |
| 追加の拒否パターン | — | — | `extraDenyPatterns` | （なし） |

- **`toolMode`**: `auto`（モデルのネイティブ関数呼び出しを優先し、非対応なら単一JSONにフォールバック）/ `function`（関数呼び出し固定）/ `json`（単一JSON固定）。関数呼び出しに対応しないモデルでは `json` を試してください。
- **`confirmMode`**: `prompt`（危険操作を確認）/ `deny`（確認せず一律拒否）。現状は設定ファイルでのみ指定できます。
- **`extraDenyPatterns`**: 組み込みの拒否ルールに加えて、コマンドに対する独自の拒否パターン（正規表現）を追加します。設定ファイルでのみ指定でき、プロジェクト設定がホーム設定を上書きします（不正な正規表現は無視され、起動時に警告します）。

### 設定ファイルの例

`./.shiroutocode.yaml`:

```yaml
endpoint: http://localhost:1234/v1
model: google/gemma-4-12b
maxSteps: 25
toolMode: auto
confirmMode: prompt
extraDenyPatterns:
  - "rm -rf /"
logLevel: info
```

## エージェントが使えるツール

| ツール | 説明 |
|---|---|
| `read_file` | ファイルを読む |
| `write_file` | ファイルを作成・更新・削除する（ワークスペース内に限定） |
| `run_command` | シェルコマンドを実行する（ワークスペースをカレントに、タイムアウト付き） |
| `git` | Git 操作を行う |
| `web_fetch` | Web上の情報を取得する（内部/メタデータ宛先はブロック） |

## 安全性（ガードレール）

ShiroutoCode は自律的にファイル編集やコマンド実行を行うため、安全制御を中核に据えています。

- **ワークスペース封じ込め** — 読み書きはワークスペース配下に限定。シンボリックリンクを解決した上で境界を判定し、外部への書き込み/削除は拒否します。
- **危険操作の確認/拒否** — 危険なコマンドやパターンは実行前に確認を求める（`prompt`）か拒否（`deny`）します。`extraDenyPatterns` で独自の拒否パターンを追加できます。
- **`.git/` 保護** — リポジトリ内部（特に `.git/hooks` 経由の任意コード実行）への書き込みは確認対象。`git --global/--system` や `hooksPath`、`git -c`、`--exec-path` など副作用の大きい操作も確認対象です。
- **SSRF対策** — `web_fetch` はループバック/リンクローカル/プライベート/クラウドメタデータ宛先をブロックします（リダイレクト先も対象）。
- **フェイルクローズ** — 設定の検証失敗時は起動を中止。非対話実行で確認できない危険操作は安全側に倒します。

> 受容済みの残存リスク（例: `run_command` の既定許可ポリシー、シンボリックリンクの TOCTOU）と脅威モデルは、ガードレール仕様書（`aidlc-docs/construction/U3-tools-guardrail/functional-design/business-rules.md` の「セキュリティ前提・残存リスク」）に記載しています。信頼できないコードを扱う場合は、隔離された作業ディレクトリでの利用を推奨します。

## ソースからのビルド

```bash
git clone https://github.com/zurustar/ShiroutoCode.git
cd ShiroutoCode
make build        # bin/shiroutocode を生成
make test         # テストを実行（race検出は make test-race）
```

`make` を使わない場合は `go build -o bin/shiroutocode ./cmd/shiroutocode`。主なターゲット: `make all`（fmt+vet+test+build）、`make cover`（カバレッジ）、`make vuln`（govulncheck）、`make cross`（クロスビルド）。

## ライセンス

[GNU General Public License v3.0](LICENSE)
