# AI-DLC State Tracking

## Project Information
- **Project Name**: ShiroutoCode
- **Project Type**: Greenfield
- **Start Date**: 2026-06-06T00:00:00Z
- **Current Stage**: CONSTRUCTION - U4 Agent Engine COMPLETE (all stages, code green, committed). NEXT: U5 CLI Frontend (final unit)
- **Session Note**: Resumed 2026-06-09. U1 done. U2 functional design generated (all recommended): hybrid tool calling (toolMode auto), single-JSON fallback, SSE chunk kinds, error taxonomy, retry policy. Artifacts at construction/U2-llm/functional-design/.
- **Units**: U1 Foundation(config,log) → U2 LLM → U3 Tools&Guardrail → U4 Agent → U5 CLI(integration+E2E)
- **Dev Convention**: TDD (test-first: red→green→refactor) across CONSTRUCTION, combined with mandated unit tests + PBT (rapid). User requested 2026-06-08.

## Execution Plan Summary
- **Stages to Execute**: Application Design, Units Planning, Units Generation, Functional Design, NFR Requirements, NFR Design, Code Generation, Build and Test
- **Stages to Skip**: Reverse Engineering (greenfield), Infrastructure Design (local-only VSCode extension, no cloud infra)

## Workspace State
- **Existing Code**: No
- **Programming Languages**: Go (core engine + CLI). TypeScript deferred for future VSCode frontend.
- **Build System**: Go modules (go build / go test). Distribution: single static binary.
- **Project Structure**: Empty (greenfield)
- **Reverse Engineering Needed**: No
- **Workspace Root**: /Users/oumi/Documents/GitHub/ShiroutoCode
- **Architecture**: Headless core + thin frontend. CLI-first (Go). VSCode extension frontend = future phase (out of current scope). (Pivot 2026-06-08)

## Code Location Rules
- **Application Code**: Workspace root (NEVER in aidlc-docs/)
- **Documentation**: aidlc-docs/ only
- **Structure patterns**: See code-generation.md Critical Rules

## Stage Progress
### 🔵 INCEPTION PHASE
- [x] Workspace Detection
- [x] Reverse Engineering (SKIPPED - greenfield)
- [x] Requirements Analysis (approved)
- [x] User Stories (approved)
- [x] Workflow Planning (approved)
- [x] Application Design — EXECUTE (approved)
- [x] Units Planning — EXECUTE (approved)
- [x] Units Generation — EXECUTE (approved)

### 🟢 CONSTRUCTION PHASE (per-unit loop; convention: TDD)
**U1 Foundation** ← CURRENT
- [x] Functional Design — EXECUTE (approved)
- [x] NFR Requirements — EXECUTE (approved)
- [x] NFR Design — EXECUTE (awaiting approval)
- [x] Infrastructure Design — SKIP (local-only, no cloud infra)
- [x] Code Generation — EXECUTE (TDD) — DONE: internal/config + internal/log, all tests green (approved)

**U2 LLM Connectivity** ← CURRENT
- [x] Functional Design — EXECUTE (approved)
- [x] NFR Requirements — EXECUTE (approved)
- [x] NFR Design — EXECUTE (approved)
- [x] Infrastructure Design — SKIP
- [x] Code Generation — EXECUTE (TDD) — DONE: internal/llm, 17 tests green incl 4 PBT, race-clean (approved)

**U3 Tools & Guardrail** (largest unit; safety core)
- [x] Functional Design — EXECUTE (approved)
- [x] NFR Requirements — EXECUTE (approved)
- [x] NFR Design — EXECUTE (approved)
- [x] Infrastructure Design — SKIP
- [x] Code Generation — EXECUTE (TDD) — DONE: internal/tools + internal/guardrail, 23 tests green incl 4 PBT, race-clean (awaiting approval)

**U4 Agent Engine** (DONE)
- [x] Functional Design (auto-approved via goal)
- [x] NFR Requirements (auto)
- [x] NFR Design (auto)
- [x] Infrastructure Design — SKIP
- [x] Code Generation — DONE: internal/agent (Runner loop, Frontend port), 6 tests green incl 1 PBT (termination), race-clean

**U5 CLI Frontend** ← NEXT (final unit: bubbletea TUI + single-shot, wires all units, E2E)
- [ ] Functional Design → NFR Req → NFR Design → [Infra SKIP] → Code Gen (TDD)
- [ ] Build and Test — EXECUTE (after U5)

### 🟡 OPERATIONS PHASE
- [ ] Operations — PLACEHOLDER

## Extension Configuration
| Extension | Enabled | Decided At |
|---|---|---|
| Security Baseline | Yes | Requirements Analysis |
| Property-Based Testing | Yes (Blocking, all rules; PBT-09: Go → rapid) | Requirements Analysis |
