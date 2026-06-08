# Component Dependency — ShiroutoCode (Application Design)

## 依存マトリクス（行が列に依存）
| ↓依存元 / 依存先→ | C1 CLI | C2 Agent | C3 LLM | C4 Tools | C5 Guardrail | C6 Config | C7 Log |
|---|---|---|---|---|---|---|---|
| **C1 CLI** | — | ✔ | | | | ✔ | ✔ |
| **C2 Agent** | ✔(Port) | — | ✔ | (経由C5) | ✔ | ✔ | ✔ |
| **C3 LLM** | | | — | | | ✔ | ✔ |
| **C4 Tools** | | | | — | | | ✔ |
| **C5 Guardrail** | ✔(Port,確認) | | | ✔ | — | ✔ | ✔ |
| **C6 Config** | | | | | | — | ✔ |
| **C7 Log** | | | | | | | — |

- C2→C1 は具体ではなく **Frontend Port インタフェース**への依存（DIで注入）。循環ではない（境界はインタフェース）。
- C2 は C4 を**直接呼ばない**。必ず C5(ToolDispatcher) 経由（Q5=A・バイパス不可）。
- C7(Log) は最下層、他に依存しない。

## 通信パターン
- **すべてプロセス内**の関数呼び出し（完全ローカル, NFR-2）。ネットワークは C3→LM Studio(HTTP/SSE) と C4.web→外部HTTP のみ。
- **イベント送出**: C2→C1 は Frontend Port のコールバック（ストリーミング表示・進行・確認）。
- **キャンセル伝播**: `context.Context` を全層に貫通（Ctrl-C → cancel, US-1.3/NFR-3）。
- **将来のVSCodeフロント**: C1 を IPC アダプタ（stdio JSON-RPC等）に差し替え、同じ Frontend Port を実装（A3、今回未実装）。

## データフロー（Mermaid）
```mermaid
flowchart TD
    User(["ユーザー (ターミナル)"])
    subgraph Frontend["Frontend 層"]
        C1["C1 CLI<br/>(Frontend Port 実装)"]
    end
    subgraph App["Application 層"]
        C2["C2 Agent Engine<br/>(Runner / Session)"]
    end
    subgraph Domain["Domain 層"]
        C5["C5 Guardrail<br/>(ToolDispatcher・単一インターセプタ)"]
        C4["C4 Tool Layer<br/>(File/Terminal/Git/Web)"]
    end
    subgraph Infra["Infrastructure 層"]
        C3["C3 LLM Client<br/>(OpenAI互換/SSE)"]
        C6["C6 Config"]
        C7["C7 Logging"]
    end
    LM(["LM Studio (ローカル)"])
    Ext(["外部Web (明示時のみ)"])

    User <--> C1
    C1 --> C2
    C2 -->|"イベント/確認 (Port)"| C1
    C2 --> C3
    C2 -->|"ToolCall"| C5
    C5 -->|"Allow後のみ"| C4
    C5 -->|"Confirm要求 (Port)"| C1
    C3 <--> LM
    C4 -->|"web.Fetch"| Ext
    C1 --> C6
    C2 --> C6
    C3 --> C6
    C5 --> C6
    C1 --> C7
    C2 --> C7
    C3 --> C7
    C4 --> C7
    C5 --> C7

    style C1 fill:#BBDEFB,stroke:#0D47A1,color:#000
    style C2 fill:#C8E6C9,stroke:#1B5E20,color:#000
    style C5 fill:#FFCDD2,stroke:#B71C1C,color:#000
    style C4 fill:#FFE0B2,stroke:#E65100,color:#000
    style C3 fill:#E1BEE7,stroke:#6A1B9A,color:#000
    style C6 fill:#F0F4C3,stroke:#827717,color:#000
    style C7 fill:#CFD8DC,stroke:#37474F,color:#000
```

## 安全性に関わる構造的不変条件
1. **ツール実行の単一窓口**: C4 へのアクセスは C5(ToolDispatcher) のみ（Q5=A）。
2. **フェイルクローズ**: C5 の判定不能/エラーは Allow にならない（SECURITY-15）。
3. **スコープ限定**: ファイル操作は C6 のワークスペースルート配下のみ。逸脱は C5 がブロック（US-5.3）。
4. **機微情報非漏洩**: C7 がマスキング、C3 のエラーは一般化（SECURITY-03/09）。
