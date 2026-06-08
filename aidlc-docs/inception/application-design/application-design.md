# Application Design — ShiroutoCode（統合設計書）

**フェーズ**: INCEPTION / Application Design
**日付**: 2026-06-08
**前提アーキテクチャ**: Go製ヘッドレスコア + 対話型CLI（CLI-first）。コアはフロント非依存（将来VSCode拡張フロントをIPCで接続）。

この文書は以下を統合する:
- [components.md](components.md) — コンポーネント定義と責務
- [component-methods.md](component-methods.md) — メソッド/インタフェースシグネチャ
- [services.md](services.md) — サービス/オーケストレーション
- [component-dependency.md](component-dependency.md) — 依存・通信・データフロー

## 設計判断サマリ（Application Design Q&A）
| # | 決定 | 採用 |
|---|---|---|
| Q1 | コンポーネント粒度 | A: 7コンポーネント |
| Q2 | ツール呼び出し方式 | C: ハイブリッド（function calling + JSONフォールバック） |
| Q3 | ファイル編集方式 | C: パッチ + 全書換の両対応 |
| Q4 | ターミナル出力 | B: stdout/stderr ストリーム表示 |
| Q5 | ガードレール適用 | A: 単一インターセプタ（バイパス不可） |
| Q6 | 状態永続化 | A: セッション内メモリのみ（永続化なし） |
| Q7 | 設計スタイル | A: レイヤード（`internal/` で境界強制） |
| Q8 | CLI操作モデル | C: REPL + 単発の両対応 |
| Q9 | その他 | モジュール `github.com/zurustar/shiroutocode` / バイナリ `shiroutocode` / CLIは標準`flag`（暫定） |

## アーキテクチャ概要
- **4レイヤー**: Frontend(CLI) → Application(Agent Engine) → Domain(Guardrail, Tools) → Infrastructure(LLM, Config, Log)。依存は上→下の一方向。
- **7コンポーネント**: C1 CLI / C2 Agent Engine / C3 LLM Client / C4 Tool Layer / C5 Guardrail / C6 Config / C7 Logging。
- **安全性の中核**: 全ツール実行は C5(ToolDispatcher) の単一窓口を必ず通過し、`Allow/Confirm/Deny` を判定。判定不能はフェイルクローズ。
- **フロント非依存**: Agent Engine は `Frontend` Port にのみ依存。CLIはその実装。将来のVSCodeフロントは同Portを別プロセス(IPC)で実装。

## 想定パッケージ構成
```
shiroutocode/
├── cmd/shiroutocode/        # エントリポイント（main）
└── internal/
    ├── cli/                 # C1 Frontend（REPL/単発、レンダリング、確認プロンプト）
    ├── agent/               # C2 Agent Engine（Runner/Session, ループ）
    ├── llm/                 # C3 LLM Client（OpenAI互換/SSE/ハイブリッド）
    ├── tools/               # C4 Tool Layer
    │   ├── file/  terminal/  git/  web/
    ├── guardrail/           # C5 Guardrail（Evaluator, ToolDispatcher）
    ├── config/              # C6 Config
    └── log/                 # C7 Logging
```
（最終構造は Units Generation / Code Generation で確定）

## 要件トレーサビリティ（抜粋）
| 要件/ストーリー | 対応コンポーネント/サービス |
|---|---|
| FR-1 CLI / US-1.1〜1.3 | C1, S1, Frontend Port（中断=context） |
| FR-2 LM Studio / US-2.1,2.2,6.1 | C3, S3, C6 |
| FR-3 ループ / US-3.1〜3.3 | C2, S1 |
| FR-4 ツール / US-4.1〜4.5 | C4（+ 必ずC5経由） |
| FR-5 承認&ガードレール / US-5.1〜5.3 | C5, S2, S4 |
| FR-6 設定 | C6 |
| NFR-3 応答性/中断 | C3(SSE), context 全層伝播 |
| NFR-4 フェイルクローズ / US-6.2 | C5, S2, C7 |
| NFR-5 ログ | C7 |
| NFR-6 テスト容易性 / PBT | DI境界、Evaluator/パス正規化/パーサ/設定優先順位（rapid） |

## 拡張コンプライアンス（Application Design段階の評価）
| ルール | 状態 | 根拠 |
|---|---|---|
| SECURITY-05 入力検証 | 関連・反映 | C6でURL検証、C5でツール引数/パス/コマンド検証 |
| SECURITY-09 ハードニング | 関連・反映 | デフォルト認証情報なし(C6)、エラー一般化(C3/C7) |
| SECURITY-10 サプライチェーン | 関連・方針 | 依存最小化（Q9 標準`flag`）、`go.sum`/`govulncheck`（Build段階） |
| SECURITY-11 セキュア設計 | 関連・中核 | C5を専用パッケージに分離、単一インターセプタ |
| SECURITY-13 完全性 | 関連・反映 | LLM出力/外部データの安全な解釈（C3パーサ、unsafe回避） |
| SECURITY-15 フェイルセーフ | 関連・中核 | C5フェイルクローズ、context/deferでのリソース解放 |
| SECURITY-01/02/04/06/07/08/12/14 | N/A | クラウド/Webサーバ/ユーザ認証/データストア前提 → 本CLIに非該当 |
| PBT-09 フレームワーク | 関連・決定 | Go → **rapid**。対象: Evaluator/パス正規化/パーサ/設定優先順位 |
| PBT その他 | 関連・後続 | 具体プロパティは Functional/NFR Design + Code Generation で定義 |

## 未決事項（後続ステージで確定）
- ハイブリッド・ツール呼び出しの具体プロトコル（JSON schema / ReActフォーマット）→ Functional Design
- ガードレール denylist/パターンの具体定義と確認UIの文言 → Functional Design
- 編集パッチ形式（unified diff 等）の詳細 → Functional Design
- CLIフレームワーク最終決定（標準`flag` vs cobra）→ 承認ゲート or NFR/Code Generation
