package guardrail

import (
	"regexp"
	"strings"
)

// ifsReplacer undoes the common $IFS / ${IFS} whitespace-obfuscation trick so
// matchers cannot be bypassed by writing e.g. "rm${IFS}-rf${IFS}~" (F-01).
// Longest pattern first so ${ifs} is consumed before the bare $ifs form.
var ifsReplacer = strings.NewReplacer("${ifs}", " ", "$ifs", " ")

// normalize lowercases, undoes IFS obfuscation, and collapses whitespace for
// robust command matching.
func normalize(c string) string {
	return strings.Join(strings.Fields(ifsReplacer.Replace(strings.ToLower(c))), " ")
}

// --- command matchers (R4 Deny) ---

func rmRootish(c string) bool {
	if !strings.Contains(c, "rm ") {
		return false
	}
	rf := strings.Contains(c, "-rf") || strings.Contains(c, "-fr") ||
		(strings.Contains(c, "--recursive") && strings.Contains(c, "--force")) ||
		(strings.Contains(c, "-r") && strings.Contains(c, "-f"))
	if !rf {
		return false
	}
	return strings.Contains(c, " /") || strings.Contains(c, " ~") || strings.Contains(c, " $home")
}

func forkBomb(c string) bool {
	return strings.Contains(c, ":(){") || strings.Contains(c, ":|:&")
}

func deviceWrite(c string) bool {
	return strings.Contains(c, "of=/dev/") || strings.Contains(c, "/dev/sd") || strings.Contains(c, "/dev/disk")
}

func mkfs(c string) bool { return strings.Contains(c, "mkfs") }

var rePower = regexp.MustCompile(`\b(shutdown|reboot|halt|poweroff)\b|\binit\s+[06]\b`)

func systemPower(c string) bool { return rePower.MatchString(c) }

// rePipeShell matches piping into a shell or any common interpreter — the
// classic "curl … | sh" remote-code pattern. Limiting it to sh/bash/zsh let
// "… | python" / "… | perl" through, so all common interpreters are covered
// (F-01).
var rePipeShell = regexp.MustCompile(`\|\s*(sudo\s+)?(sh|bash|zsh|dash|ksh|fish|python[0-9.]*|perl|ruby|node|php)\b`)

func pipeToShell(c string) bool { return rePipeShell.MatchString(c) }

// --- command matchers (R5 Confirm) ---

func sudo(c string) bool { return regexp.MustCompile(`\bsudo\b`).MatchString(c) }

func recursivePerm(c string) bool {
	return regexp.MustCompile(`\bch(mod|own)\b`).MatchString(c) && (strings.Contains(c, "-r") || strings.Contains(c, "--recursive"))
}

// --- git matchers (R6) ---

func gitDestructive(c string) bool {
	patterns := []string{
		`git\s+push\b.*(--force|-f)\b`,
		`git\s+reset\b.*--hard`,
		`filter-branch`,
		`git\s+clean\b.*-[a-z]*f`,
		`--amend`,
		`git\s+rebase\b`,
	}
	for _, p := range patterns {
		if regexp.MustCompile(p).MatchString(c) {
			return true
		}
	}
	return false
}

func gitPush(c string) bool { return regexp.MustCompile(`git\s+push\b`).MatchString(c) }

// gitGlobalSideEffect matches git operations whose effect reaches outside the
// workspace or can establish a code-execution hook: global/system config
// writes, core.hooksPath changes, inline -c overrides, and --exec-path (F-07).
// Plain repo-local `git config user.email …` is intentionally not matched.
var (
	reGitGlobalConfig = regexp.MustCompile(`git\s+config\b.*(--global|--system)`)
	reGitInlineConfig = regexp.MustCompile(`git\s+-c\b`)
)

func gitGlobalSideEffect(c string) bool {
	return reGitGlobalConfig.MatchString(c) ||
		reGitInlineConfig.MatchString(c) ||
		strings.Contains(c, "hookspath") ||
		strings.Contains(c, "--exec-path")
}

// Rule is one ordered evaluation rule.
type Rule struct {
	Kind     ActionKind
	Match    func(a Action) bool
	Decision DecisionKind
	Reason   string
}

func cmd(m func(string) bool) func(Action) bool {
	return func(a Action) bool { return m(normalize(a.CommandLine)) }
}

// defaultRules returns the built-in ordered rule table. Deny rules precede
// Confirm rules; the first match wins (R4-R7).
func defaultRules() []Rule {
	return []Rule{
		// Command — Deny
		{Command, cmd(rmRootish), Deny, "ルート/ホーム配下を再帰削除する危険なコマンドです"},
		{Command, cmd(forkBomb), Deny, "fork爆弾の可能性があります"},
		{Command, cmd(deviceWrite), Deny, "デバイスへの直接書き込みは危険です"},
		{Command, cmd(mkfs), Deny, "ファイルシステム作成は破壊的です"},
		{Command, cmd(systemPower), Deny, "システムの電源/再起動操作です"},
		{Command, cmd(pipeToShell), Deny, "ダウンロード結果を直接シェルに流す実行は危険です"},
		// Command — Confirm
		{Command, cmd(sudo), Confirm, "権限昇格(sudo)を伴います"},
		{Command, cmd(recursivePerm), Confirm, "権限の再帰変更は影響が広範です"},
		// Git — Confirm (destructive/history-affecting and remote push)
		{GitOp, cmd(gitDestructive), Confirm, "強制push/履歴改変/ハード破棄など取り消し困難なgit操作です"},
		{GitOp, cmd(gitGlobalSideEffect), Confirm, "グローバル/システム設定・hooksPath・インラインconfigなどワークスペース外に影響するgit操作です"},
		{GitOp, cmd(gitPush), Confirm, "リモートへのpushです"},
		// Web — Deny non-http(s)
		{WebFetch, func(a Action) bool { return !isHTTPScheme(a.URL) }, Deny, "http(s)以外のURLは許可されていません"},
	}
}

func isHTTPScheme(raw string) bool {
	r := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(r, "http://") || strings.HasPrefix(r, "https://")
}
