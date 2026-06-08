# Components — ShiroutoCode (Application Design)

**前提**: Go製ヘッドレスコア + 対話型CLI（CLI-first）。コアはフロント非依存。
**設計判断（Application Design Q&A 2026-06-08）**: Q1=A(7コンポーネント) / Q2=C(ツール呼び出しハイブリッド) / Q3=C(編集はパッチ+全書換) / Q4=B(コマンド出力ストリーム) / Q5=A(ガードレール単一インターセプタ) / Q6=A(状態はメモリのみ) / Q7=A(レイヤード) / Q8=C(REPL+単発)。

## レイヤリング（Q7=A）
```
CLI (Frontend)        → Application（Agent Engine / orchestration）
                      → Domain（Guardrail, Tool 抽象, ドメインモデル）
                      → Infrastructure（LLM Client, OS/exec, Git, HTTP, Config, Logging）
```
依存は上→下の一方向。Goの `internal/` でパッケージ境界を強制する。

## フロント非依存の要（Frontend Port）
Agent Engine は具体的なCLIに依存せず、**Frontend Port インタフェース**（イベント送出 + 確認要求）にのみ依存する。CLIはこのPortの実装。将来のVSCode拡張フロントは同Portを **IPC（stdio JSON-RPC等）越し**に実装する（今回未実装）。

---

## C1. CLI Frontend
- **パッケージ**: `cmd/shiroutocode`（エントリ）, `internal/cli`
- **目的**: ユーザーとの対話接点。Frontend Port の実装。
- **責務**:
  - 引数/フラグ解釈、単発実行(`shiroutocode "指示"`)と対話REPLの両対応（Q8=C）
  - エージェントのイベント（メッセージ/思考/ツール呼び出し/結果/ステップ）を時系列で逐次レンダリング（区別可能な表示）
  - Ctrl-C を捕捉し context をキャンセル（中断, US-1.3）
  - ガードレールの確認要求をユーザーに提示し諾否を返す（US-5.2）。**非TTY/非対話時は確認不能 → 安全側で停止**
  - 端末がカラー/TTY非対応でもプレーン出力にフォールバック
- **インタフェース（提供）**: `Frontend` Port（C2が利用）を実装
- **依存**: Config(C6), Agent Engine(C2), Logging(C7)

## C2. Agent Engine
- **パッケージ**: `internal/agent`
- **目的**: 自律エージェントの中核。plan→act→observe ループの実行（FR-3）。
- **責務**:
  - 1タスク=1セッションを実行。会話履歴を**メモリ上で**保持（Q6=A、永続化なし）
  - LLM Client を呼び、応答からツール呼び出しを解釈（ハイブリッド Q2=C）
  - ツール実行は **ToolDispatcher（Guardrail内蔵）経由**でのみ行う
  - ステップ管理・終了条件（完了判定 / 最大ステップ / 中断）（US-3.1/3.2/3.3）
  - 進行イベントと最終要約を Frontend Port へ送出
- **インタフェース（提供）**: `Runner.Run(ctx, task) (Result, error)`
- **依存**: LLM Client(C3), Tool Dispatcher(=Guardrail C5 + Tool Layer C4), Frontend Port(C1), Logging(C7)

## C3. LLM Client
- **パッケージ**: `internal/llm`
- **目的**: LM Studio（OpenAI互換REST）連携（FR-2）。
- **責務**:
  - `/v1/chat/completions` 呼び出し、**SSEストリーミング**受信・パース（NFR-3）
  - **ハイブリッド対応（Q2=C）**: function calling（`tools`）対応モデルではそれを使用、非対応時はプロンプト＋JSON/ReActパースにフォールバック。能力判定/モード選択を内包
  - 接続失敗・タイムアウト時に分かりやすいエラー情報を返す（US-6.1、内部情報は露出しない SECURITY-09）
- **インタフェース（提供）**: `Complete(ctx, req) (Stream, error)` / `Capabilities()`
- **依存**: Config(C6), Logging(C7), 標準 `net/http`

## C4. Tool Layer
- **パッケージ**: `internal/tools`（`file`, `terminal`, `git`, `web` サブパッケージ）
- **目的**: エージェントが実行できる操作の実装（FR-4）。
- **責務**:
  - 共通 `Tool` インタフェース（名前 / 入力スキーマ / 実行）
  - **FileRead**（FR-4.1）、**FileMutate**: create/edit/delete、編集はパッチ+全書換の両対応（Q3=C, FR-4.2）
  - **Terminal**: `os/exec`、stdout/stderr を**ストリーム**で返す（Q4=B, FR-4.3）
  - **Git**: commit/branch 等（FR-4.5）
  - **Web**: 明示的取得のみ（FR-4.4, NFR-2）
  - ツールレジストリ（名前→Tool、LLMへ渡すスキーマ生成）
- **インタフェース（提供）**: `Tool`（`Name() string`, `Spec() ToolSpec`, `Execute(ctx, args) (ToolResult, error)`）, `Registry`
- **依存**: Logging(C7), OS/Git/HTTP（Infrastructure）。**ガードレール判定はC5が外側で実施**（ツール自身は判定を持たない）

## C5. Guardrail
- **パッケージ**: `internal/guardrail`
- **目的**: セーフティ制御の中核（FR-5, SECURITY-11）。**全ツール呼び出しを通す単一インターセプタ（Q5=A）**。
- **責務**:
  - すべてのツール実行直前に評価し `Allow / Confirm / Deny` を決定
  - **ワークスペーススコープ限定**（パス正規化、シンボリックリンク回避）（US-5.3）
  - **危険コマンド/操作のパターン検出**（denylist: `rm -rf /`、強制push、履歴改変、認証情報外部送信、攻撃的アクセス等）（US-5.2）
  - **フェイルクローズ**: 判定不能・エラー時は危険側に倒さず Confirm/Deny（SECURITY-15, US-6.2）
  - Confirm 時は Frontend Port 経由で明示確認を要求（非対話なら停止）
- **インタフェース（提供）**: `ToolDispatcher`（`Dispatch(ctx, tool, args) (ToolResult, error)` … 内部で評価→確認→C4実行）, `Evaluate(action) Decision`
- **依存**: Config(C6, ポリシー/ワークスペースルート), Tool Layer(C4), Frontend Port(C1, 確認), Logging(C7)

## C6. Config
- **パッケージ**: `internal/config`
- **目的**: 構成管理（FR-6）。
- **責務**:
  - 優先順位 **フラグ > 環境変数 > 設定ファイル > 既定** で統合
  - 項目: LM Studioエンドポイント/モデル、最大ステップ、ガードレール挙動、ワークスペースルート
  - 検証（URL形式 SECURITY-05）、**デフォルト認証情報を持たない**（SECURITY-09）
- **インタフェース（提供）**: `Load(args, env) (Config, error)`
- **依存**: Logging(C7)

## C7. Observability / Logging
- **パッケージ**: `internal/log`
- **目的**: 構造化ログと可観測性（NFR-5）。
- **責務**:
  - 構造化ログ（timestamp / correlation ID / level / message）（SECURITY-03）
  - **機微情報（トークン/PII）のマスキング**
- **インタフェース（提供）**: `Logger`（`Info/Warn/Error`, `With(fields)`）
- **依存**: なし（最下層）

---

## Q9 デフォルト（承認ゲートで上書き可）
- **モジュールパス**: `github.com/zurustar/shiroutocode`（gitユーザー zurustar より）
- **バイナリ名**: `shiroutocode`
- **CLIフレームワーク**: MVPは**標準 `flag`**（サプライチェーン最小化 SECURITY-10）。サブコマンドが増えたら `cobra` 等を後で検討。
- **参考設計**: Claude Code / aider（ヘッドレスコア + 薄フロント、ツール駆動ループ）
