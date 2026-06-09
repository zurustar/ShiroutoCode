package llm

import (
	"encoding/json"
	"testing"

	"pgregory.net/rapid"
)

// R3 (PBT): a well-formed tool/final JSON object round-trips through the parser.
func TestJSONToolRoundTripPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		isTool := rapid.Bool().Draw(rt, "isTool")
		if isTool {
			name := rapid.StringMatching(`[a-z_]{1,10}`).Draw(rt, "name")
			keys := rapid.SliceOfDistinct(rapid.StringMatching(`[a-z]{1,5}`), func(s string) string { return s }).Draw(rt, "keys")
			args := map[string]string{}
			for _, k := range keys {
				args[k] = rapid.StringMatching(`[a-zA-Z0-9 ]{0,8}`).Draw(rt, "v_"+k)
			}
			obj := map[string]any{"tool": name, "args": args}
			raw, _ := json.Marshal(obj)
			call, final, err := parseJSONTool(string(raw))
			if err != nil {
				rt.Fatalf("parse tool: %v", err)
			}
			if final != "" || call == nil {
				rt.Fatalf("expected tool call, got final=%q call=%v", final, call)
			}
			if call.Name != name {
				rt.Fatalf("name = %q want %q", call.Name, name)
			}
			for k, v := range args {
				if got, _ := call.Args[k].(string); got != v {
					rt.Fatalf("arg %q=%q want %q", k, got, v)
				}
			}
		} else {
			text := rapid.String().Draw(rt, "final")
			obj := map[string]any{"final": text}
			raw, _ := json.Marshal(obj)
			call, final, err := parseJSONTool(string(raw))
			if err != nil {
				rt.Fatalf("parse final: %v", err)
			}
			if call != nil {
				rt.Fatalf("expected final, got call %v", call)
			}
			if final != text {
				rt.Fatalf("final = %q want %q", final, text)
			}
		}
	})
}

func TestJSONToolToleratesFencesAndProse(t *testing.T) {
	in := "Sure!\n```json\n{\"tool\":\"read_file\",\"args\":{\"path\":\"a.txt\"}}\n```\n"
	call, final, err := parseJSONTool(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if final != "" || call == nil || call.Name != "read_file" {
		t.Fatalf("got call=%v final=%q", call, final)
	}
	if call.Args["path"] != "a.txt" {
		t.Errorf("args=%v", call.Args)
	}
}

func TestJSONToolUndecodableIsError(t *testing.T) {
	for _, in := range []string{"", "no json here", "{}", `{"foo":1}`} {
		if _, _, err := parseJSONTool(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}
