package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// agentsFileName is the conventional project-instructions file (agents.md).
const agentsFileName = "AGENTS.md"

// maxAgentsDocBytes caps how much of AGENTS.md is injected into the system
// prompt, so an oversized file cannot blow the model's context window.
const maxAgentsDocBytes = 32 * 1024

// loadAgentsDoc reads AGENTS.md from the workspace root, following the
// https://agents.md convention. It returns the (possibly truncated) content
// and whether a non-empty file was found. Missing/unreadable/empty files
// return ("", false).
func loadAgentsDoc(workspace string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(workspace, agentsFileName))
	if err != nil {
		return "", false
	}
	doc := strings.TrimSpace(string(data))
	if doc == "" {
		return "", false
	}
	truncated := false
	if len(doc) > maxAgentsDocBytes {
		doc = doc[:maxAgentsDocBytes]
		truncated = true
	}
	if truncated {
		doc += "\n\n[... AGENTS.md truncated ...]"
	}
	return doc, true
}

// composeSystemPrompt appends project instructions from AGENTS.md to the base
// system prompt. When doc is empty the base prompt is returned unchanged.
func composeSystemPrompt(base, doc string) string {
	if strings.TrimSpace(doc) == "" {
		return base
	}
	return base + "\n\n# Project instructions (AGENTS.md)\n" +
		"The repository provides the following project-specific instructions. " +
		"Follow them; if they conflict with the above, prefer these for project-specific decisions.\n\n" +
		doc
}
