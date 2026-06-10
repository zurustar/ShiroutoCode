# End-to-End Test Instructions（実 LM Studio）

## Purpose
実モデルに対して「自然言語の指示一つで安全にタスク完遂」（US-3.1 / MVPの中核）を検証する。**LM Studio 起動が必要**。

## 前提
1. LM Studio で **Local Server を起動**（OpenAI互換, 既定 `http://localhost:1234/v1`）。
2. **モデルをロード**（tool/function calling 対応の instruct 系が望ましい。非対応でも JSON フォールバックで動作）。
3. ビルド済みバイナリ: `make build` → `bin/shiroutocode`。

## 推奨手順（安全な一時ワークスペースで）
```bash
WS=$(mktemp -d)
export SHIROUTO_MODEL="<LM Studioのモデル名>"
export SHIROUTO_ENDPOINT="http://localhost:1234/v1"
export SHIROUTO_WORKSPACE="$WS"

# E2E-1: 単一ファイル作成（自動承認の通常操作）
./bin/shiroutocode "$WS に hello.txt を作り、本文に 'Hello from ShiroutoCode' と書いて"
cat "$WS/hello.txt"      # → 期待文字列

# E2E-2: マルチファイル編集（MVP中核, US-3.1）
#   例: 簡単な go ファイル2つを作り、関数名を一括変更させる 等
./bin/shiroutocode "$WS に greet.go と main.go を作り、main から greet を呼ぶようにして"
ls "$WS"; (cd "$WS" && gofmt -l .)

# E2E-3: ガードレール（危険操作のブロック / 確認, US-5.2）
#   対話モード（端末で実行）で危険コマンドを依頼 → 確認プロンプト or 拒否を確認
./bin/shiroutocode            # REPL(TUI): "ワークスペースの外の /etc/hosts を削除して" 等 → ブロック/確認

# E2E-4: 中断（US-1.3）
#   長い処理中に Ctrl+C → 安全停止
```

## 期待結果（チェックリスト）
- [ ] 指示通りのファイルが**ワークスペース内に**生成/編集される（US-3.1/4.2）
- [ ] 実行アクション（ツール呼び出し・結果）が時系列で表示される（US-1.2）
- [ ] ストリーミングで応答が逐次表示される（US-2.2）
- [ ] ワークスペース外への破壊的操作が **Deny / 確認要求** される（US-5.2/5.3）
- [ ] 最大ステップ到達で停止する（US-3.3）
- [ ] LM Studio を止めると分かりやすい接続エラー（US-6.1）

## 実施済み E2E 結果（2026-06-10, 実モデル: `google/gemma-4-12b` via LM Studio）
| シナリオ | コマンド要旨 | 結果 |
|---|---|---|
| 単一ファイル作成（US-3.1） | "hello.txt に Hello… を作成" | ✅ 2ステップで write_file→完了、内容一致 |
| マルチファイル（US-3.1中核） | "a.txt=alpha, b.txt=beta 作成し読み戻す" | ✅ 5ステップ（write×2, read×2, final）、両ファイル一致 |
| スコープ防御（US-5.3） | "/tmp/外部.txt を作成" | ✅ ガードレールが Deny、ファイル未作成、モデルに制限通知 |
| 接続失敗（US-6.1） | LM Studio停止時 | ✅ リトライ後「接続できません」案内・exit≠0 |

### 重要な実地修正（E2Eで発見）
- **ツールスキーマに `properties` が無く function calling が HTTP 400** になっていた → 各ツールに JSON Schema（`internal/tools/schema.go`）を追加して解消。これにより Gemma 4 でも `--tool-mode auto`（function calling）が機能。
- Gemma 4 は OpenAI 互換 function calling を LM Studio 経由で利用可（独自 `<|tool_call|>` トークンはサーバが tool_calls に変換）。`--tool-mode json` 単独だとモデルが独自トークンを出しパース不可だったため、**auto/function 推奨**。

## 備考
- モデルの能力により tool 呼び出し精度が変わる（function calling対応モデル推奨）。
- 推奨実行: `--tool-mode auto`（既定）。function calling非対応モデルでは `--tool-mode json`（プロンプト規約）。
