# Execution Plan — ShiroutoCode

## Detailed Analysis Summary

### Transformation Scope
- **Project Type**: Greenfield（ゼロからの新規開発）→ Brownfield固有の分析（変換スコープ/コンポーネント関係/モジュール調整）は **N/A**。
- **Primary Changes**: TypeScript製VSCode拡張を新規構築。サイドバーChat UI（Webview）＋自律エージェントループ＋LM Studio（OpenAI互換REST）連携＋ツール実行層（File/Terminal/Git/Web）＋セーフティガードレール。
- **Related Components**: 拡張ホスト（Extension Host）、Webview UI、エージェントエンジン、LLMクライアント、ツール群、ガードレール、設定。

### Change Impact Assessment
- **User-facing changes**: Yes — サイドバーChat UI、思考/アクション可視化、中断操作、エラーUX。
- **Structural changes**: Yes — システム全体を新規にアーキテクチャ設計（多コンポーネント）。
- **Data model changes**: Yes（新規）— 会話/メッセージ、ツール呼び出し、エージェント状態、設定スキーマ。永続DBは持たない（ローカルのみ）。
- **API changes**: Yes（外部依存）— LM Studio OpenAI互換 `/v1/chat/completions`（SSEストリーミング）への変換層。
- **NFR impact**: Yes — 性能（ストリーミング/非ブロッキング）、セキュリティ（ガードレール=中核）、信頼性（フェイルクローズ）、可観測性、テスト容易性（PBT）。

### Risk Assessment
- **Risk Level**: High — 自律エージェントがファイル編集・コマンド実行・Git・Webを**自動承認**で行うため、ガードレールの正しさが安全性の要。複数コンポーネントの協調も必要。
- **Rollback Complexity**: Easy（開発時）— グリーンフィールドのためコード/ドキュメントの巻き戻しは容易。ただし実行時のワークスペース変更は破壊的になりうる（→ガードレールとフェイルクローズで緩和）。
- **Testing Complexity**: Complex — エージェントループ、ガードレール判定、API変換層は単体＋プロパティベーステスト（fast-check）対象。

## Workflow Visualization

```mermaid
flowchart TD
    Start(["User Request"])

    subgraph INCEPTION["🔵 INCEPTION PHASE"]
        WD["Workspace Detection<br/><b>COMPLETED</b>"]
        RE["Reverse Engineering<br/><b>SKIP (greenfield)</b>"]
        RA["Requirements Analysis<br/><b>COMPLETED</b>"]
        US["User Stories<br/><b>COMPLETED</b>"]
        WP["Workflow Planning<br/><b>IN PROGRESS</b>"]
        AD["Application Design<br/><b>EXECUTE</b>"]
        UP["Units Planning<br/><b>EXECUTE</b>"]
        UG["Units Generation<br/><b>EXECUTE</b>"]
    end

    subgraph CONSTRUCTION["🟢 CONSTRUCTION PHASE"]
        FD["Functional Design<br/><b>EXECUTE</b>"]
        NFRA["NFR Requirements<br/><b>EXECUTE</b>"]
        NFRD["NFR Design<br/><b>EXECUTE</b>"]
        ID["Infrastructure Design<br/><b>SKIP</b>"]
        CG["Code Generation<br/>(Planning + Generation)<br/><b>EXECUTE</b>"]
        BT["Build and Test<br/><b>EXECUTE</b>"]
    end

    subgraph OPERATIONS["🟡 OPERATIONS PHASE"]
        OPS["Operations<br/><b>PLACEHOLDER</b>"]
    end

    Start --> WD
    WD --> RA
    RA --> US
    US --> WP
    WP --> AD
    AD --> UP
    UP --> UG
    UG --> FD
    FD --> NFRA
    NFRA --> NFRD
    NFRD --> CG
    CG --> BT
    BT --> End(["Complete"])

    style WD fill:#4CAF50,stroke:#1B5E20,stroke-width:3px,color:#fff
    style RA fill:#4CAF50,stroke:#1B5E20,stroke-width:3px,color:#fff
    style US fill:#4CAF50,stroke:#1B5E20,stroke-width:3px,color:#fff
    style WP fill:#4CAF50,stroke:#1B5E20,stroke-width:3px,color:#fff
    style CG fill:#4CAF50,stroke:#1B5E20,stroke-width:3px,color:#fff
    style BT fill:#4CAF50,stroke:#1B5E20,stroke-width:3px,color:#fff
    style AD fill:#FFA726,stroke:#E65100,stroke-width:3px,stroke-dasharray: 5 5,color:#000
    style UP fill:#FFA726,stroke:#E65100,stroke-width:3px,stroke-dasharray: 5 5,color:#000
    style UG fill:#FFA726,stroke:#E65100,stroke-width:3px,stroke-dasharray: 5 5,color:#000
    style FD fill:#FFA726,stroke:#E65100,stroke-width:3px,stroke-dasharray: 5 5,color:#000
    style NFRA fill:#FFA726,stroke:#E65100,stroke-width:3px,stroke-dasharray: 5 5,color:#000
    style NFRD fill:#FFA726,stroke:#E65100,stroke-width:3px,stroke-dasharray: 5 5,color:#000
    style RE fill:#BDBDBD,stroke:#424242,stroke-width:2px,stroke-dasharray: 5 5,color:#000
    style ID fill:#BDBDBD,stroke:#424242,stroke-width:2px,stroke-dasharray: 5 5,color:#000
    style OPS fill:#FFF59D,stroke:#F57F17,stroke-width:2px,color:#000
    style Start fill:#CE93D8,stroke:#6A1B9A,stroke-width:3px,color:#000
    style End fill:#CE93D8,stroke:#6A1B9A,stroke-width:3px,color:#000

    linkStyle default stroke:#333,stroke-width:2px
```

## Phases to Execute

### 🔵 INCEPTION PHASE
- [x] Workspace Detection (COMPLETED)
- [x] Reverse Engineering (SKIPPED)
  - **Rationale**: グリーンフィールド。既存コードなし。
- [x] Requirements Analysis (COMPLETED)
- [x] User Stories (COMPLETED)
- [x] Execution Plan (IN PROGRESS)
- [ ] Application Design — **EXECUTE**
  - **Rationale**: 新規システムであり、複数の新規コンポーネント（Webview UI、エージェントエンジン、LLMクライアント、各ツール、ガードレール、設定）の責務・メソッド・依存関係・ビジネスルールの定義が必要。
- [ ] Units Planning — **EXECUTE**
  - **Rationale**: システム全体規模で関心事が明確に分かれる（UI / エージェント中核 / LLM連携 / ツール / ガードレール）。構造的に作業単位へ分解することで設計・実装・テストを安全に進められる。
- [ ] Units Generation — **EXECUTE**
  - **Rationale**: 上記の分解を具体的な units of work として確定し、Construction を unit 単位で回す。

### 🟢 CONSTRUCTION PHASE（各 unit ごとにループ）
- [ ] Functional Design — **EXECUTE**
  - **Rationale**: エージェントループ（plan→act→observe / 終了条件）、ガードレール判定ロジック、OpenAI互換API変換・SSE処理など、複雑なビジネスロジックとデータモデルの詳細設計が必要。
- [ ] NFR Requirements — **EXECUTE**
  - **Rationale**: 性能（ストリーミング/UI非ブロッキング）、セキュリティ（ガードレール=中核、Security Baseline拡張がBlocking）、信頼性（フェイルクローズ）、可観測性、テスト容易性（PBT拡張がBlocking）の要件確定が必要。技術スタックはTypeScriptで確定済みだがNFR観点は残る。
- [ ] NFR Design — **EXECUTE**
  - **Rationale**: NFR Requirementsを実装パターン（非同期/キャンセル、構造化ログ、機微情報マスキング、フェイルクローズ、PBT戦略）へ落とし込む。
- [ ] Infrastructure Design — **SKIP**
  - **Rationale**: クラウド/サーバ/データストアを持たない完全ローカルのVSCode拡張（NFR-2）。デプロイ対象インフラなし。配布（vsce パッケージング / Marketplace）はBuild and Testおよび将来のOperationsで扱う。
- [ ] Code Generation — **EXECUTE (ALWAYS)**
  - **Rationale**: 実装計画と実コード（拡張本体・UI・エージェント・ツール・ガードレール・テスト）の生成が必要。
- [ ] Build and Test — **EXECUTE (ALWAYS)**
  - **Rationale**: ビルド（tsc/bundler）、単体テスト、プロパティベーステスト（fast-check）、統合テスト、vsceパッケージングの検証が必要。

### 🟡 OPERATIONS PHASE
- [ ] Operations — **PLACEHOLDER**
  - **Rationale**: 将来のデプロイ/監視ワークフロー用プレースホルダ。Marketplace公開の運用はここで将来扱う。

## Extension Enforcement（有効化済み拡張の適用方針）
| Extension | Enabled | 適用方針 |
|---|---|---|
| Security Baseline | Yes (Blocking) | 各ステージで関連ルールを評価。特にガードレール（SECURITY-11）、入力検証（SECURITY-05）、フェイルセーフ（SECURITY-15）、サプライチェーン（SECURITY-10）、ハードニング（SECURITY-09）、完全性（SECURITY-13）。クラウド/認証前提のルールはN/A判定（根拠を各ステージで明記）。 |
| Property-Based Testing | Yes (Blocking) | TypeScript → fast-check（PBT-09）。エージェントループ・ガードレール判定・API変換層など純粋ロジックにPBTを適用。Functional/NFR Design および Code Generation で具体化。 |

## Package / Unit Change Sequence
グリーンフィールドのため厳密な依存順は Units Generation で確定するが、想定される論理的順序：
1. **拡張シェル & 設定基盤**（Extension activation、settings、ログ基盤）— 他全unitの土台
2. **LLMクライアント**（OpenAI互換REST / SSEストリーミング、エラーハンドリング）
3. **ツール層**（File / Terminal / Git / Web）+ **ガードレール**（横断的安全制御）
4. **エージェントエンジン**（plan→act→observe、終了条件、中断）— 上記を統合
5. **Webview Chat UI**（入力・履歴・可視化・中断・エラーUX）— ユーザー接点

## Estimated Timeline
- **Total Stages to Execute (残り)**: INCEPTION 3（Application Design / Units Planning / Units Generation）+ CONSTRUCTION 各unit×（Functional / NFR Req / NFR Design / Code Gen）+ Build and Test。
- **Estimated Duration**: 反復的（unit数に依存）。Units Generation で units 数が確定後に精緻化。

## Success Criteria
- **Primary Goal**: 自然言語の指示一つで、ローカルLLM（LM Studio）を用いて複数ファイルにまたがる変更を**安全に自動完遂**できるVSCode拡張のMVP（US-3.1）。
- **Key Deliverables**:
  - サイドバーChat UI（履歴/思考/アクション可視化、中断）
  - LM Studio連携（設定可能エンドポイント、SSEストリーミング）
  - 自律エージェントループ（終了条件・最大ステップ）
  - ツール群（File/Terminal/Git/Web）
  - セーフティガードレール（専用モジュール、フェイルクローズ）
  - 単体テスト + プロパティベーステスト
- **Quality Gates**:
  - Security Baseline（Blocking）適用ルールに非準拠がないこと
  - PBT（Blocking）が対象ロジックに適用されていること
  - フェイルクローズ／ワークスペーススコープ限定が検証可能なこと
