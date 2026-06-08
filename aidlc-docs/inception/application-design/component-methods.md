# Component Methods — ShiroutoCode (Application Design)

> 高レベルのメソッド/インタフェースシグネチャ（Go風）。**詳細なビジネスルールは Functional Design（per-unit, CONSTRUCTION）で確定**。型は方向性を示す概略。

## 共通ドメイン型（概略）
```go
// 1タスクの実行指示
type Task struct {
    Prompt    string
    Workspace string // 絶対パス
}

// エージェント実行結果
type Result struct {
    Status       Status   // Completed / Stopped(MaxSteps) / Aborted / Failed
    Summary      string   // 変更点の要約
    ChangedFiles []string
    Steps        int
}

// ツール呼び出し（LLM出力を正規化したもの）
type ToolCall struct {
    Tool string
    Args map[string]any
}

type ToolResult struct {
    Output   string
    ExitCode int            // Terminal用
    Stream   <-chan []byte  // ストリーム出力（Q4=B, 任意）
    Err      error
}

// ガードレール判定
type Decision int // Allow / Confirm / Deny
type Action struct {
    Tool string
    Args map[string]any
    Kind ActionKind // FileWrite/FileDelete/Command/GitPush/Web ...
}
```

## C1. CLI Frontend — `Frontend` Port を実装
```go
// Agent Engine が依存する唯一のフロント抽象（VSCode版も同じPortをIPC越しに実装）
type Frontend interface {
    OnAgentMessage(text string)        // LLMの応答テキスト（ストリーム断片可）
    OnThought(text string)             // 思考過程
    OnToolCall(call ToolCall)          // ツール呼び出し開始
    OnToolResult(call ToolCall, res ToolResult)
    OnStep(current, max int)           // ループ進行（US-3.2）
    Confirm(ctx context.Context, a Action, reason string) (bool, error) // 危険操作の明示確認（US-5.2）。非対話時は error/false で安全停止
}

// CLIエントリ
func Run(ctx context.Context, args []string, env map[string]string) int // 終了コード
func newREPL(cfg config.Config) *repl   // 対話モード（Q8=C）
func runOnce(cfg config.Config, prompt string) Result // 単発（Q8=C）
```

## C2. Agent Engine
```go
type Runner interface {
    Run(ctx context.Context, task Task) (Result, error)
}
// 依存を注入して生成
func NewRunner(llm llm.Client, disp guardrail.ToolDispatcher, fe Frontend, log log.Logger, cfg config.Config) Runner

// 内部（概略）: ループ 1ステップ
//   1) LLMへ会話+ツールスキーマ送信 → 応答（テキスト or ToolCall）
//   2) ToolCallあれば disp.Dispatch(...) 経由で実行 → 観測を会話へ追加
//   3) 完了判定 / max steps / ctx.Done() を評価
```

## C3. LLM Client
```go
type Client interface {
    Capabilities(ctx context.Context) (Caps, error) // function calling対応可否など（Q2=C）
    Complete(ctx context.Context, req Request) (Stream, error)
}
type Request struct {
    Messages []Message
    Tools    []ToolSpec // function calling時に提示
    Stream   bool
}
type Stream interface { // SSE
    Recv() (Chunk, error) // io.EOF で終了。Chunkはテキスト断片 or ToolCall断片
    Close() error
}
func New(cfg config.Config, log log.Logger) Client
```

## C4. Tool Layer
```go
type Tool interface {
    Name() string
    Spec() ToolSpec // LLMへ渡すスキーマ（引数定義）
    Execute(ctx context.Context, args map[string]any) (ToolResult, error)
}
type Registry interface {
    Register(t Tool)
    Get(name string) (Tool, bool)
    Specs() []ToolSpec
}

// 主な実装（メソッドは Execute に集約）
//  file.Read / file.Mutate(create|edit(patch|full)|delete)   (Q3=C)
//  terminal.Exec  → ToolResult.Stream で stdout/stderr 逐次   (Q4=B)
//  git.Op(commit|branch|...)        web.Fetch(url)
```

## C5. Guardrail（単一インターセプタ, Q5=A）
```go
type ToolDispatcher interface {
    // 全ツール実行の唯一の入口。内部で Evaluate → (必要なら) Frontend.Confirm → Registry経由でExecute
    Dispatch(ctx context.Context, call ToolCall) (ToolResult, error)
}
type Evaluator interface {
    Evaluate(a Action) (Decision, reason string) // Allow/Confirm/Deny。判定不能はConfirm/Deny（フェイルクローズ）
}
func NewDispatcher(reg tools.Registry, ev Evaluator, fe Frontend, cfg config.Config, log log.Logger) ToolDispatcher

// スコープ検証（US-5.3）
func withinWorkspace(root, target string) bool // パス正規化 + シンボリックリンク解決
```

## C6. Config
```go
type Config struct {
    Endpoint    string // 既定 http://localhost:1234/v1
    Model       string
    MaxSteps    int
    Workspace   string
    Guardrail   GuardrailPolicy
}
func Load(args []string, env map[string]string) (Config, error) // 優先順位: flag > env > file > default。URL検証(SECURITY-05)
```

## C7. Logging
```go
type Logger interface {
    Info(msg string, kv ...any)
    Warn(msg string, kv ...any)
    Error(msg string, kv ...any)
    With(kv ...any) Logger // correlation ID 等
}
func New(level string) Logger // 出力時に機微情報をマスク(SECURITY-03)
```
