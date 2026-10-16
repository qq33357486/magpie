package gateway

import (
	"encoding/json"
	"testing"
)

// A model told the conversation again the way the gateway tells a run of it
// (renderParts, claude_subscription.go) writes its calls the way it reads
// them made: "[tool call name id=… args={…}]", the name as the bridge wrote
// it. Each block that names an offered tool, with arguments that read, is a
// call; anything else leaves the text as it came.
func TestMagpieCalls(t *testing.T) {
	names := map[string]bool{"read": true, "bash": true}
	cases := []struct {
		name  string
		in    string
		calls []string // each "name args", "" for none
		ok    bool
	}{
		{"one", `[tool call mcp__magpie__read id=c1 args={"path":"/tmp/x"}]`,
			[]string{`read {"path":"/tmp/x"}`}, true},
		{"no id, no prefix", `[tool call bash args={"cmd":"ls"}]`,
			[]string{`bash {"cmd":"ls"}`}, true},
		{"two", `[tool call read id=a args={}][tool call mcp__magpie__bash id=b args={"cmd":"ls"}]`,
			[]string{`read {}`, `bash {"cmd":"ls"}`}, true},
		{"nested arguments", `[tool call read id=c2 args={"path":"/x","o":{"deep":[1,2]}}]`,
			[]string{`read {"o":{"deep":[1,2]},"path":"/x"}`}, true},
		{"a tool not offered", `[tool call ghost id=c1 args={}]`, nil, false},
		{"words after the arguments", `[tool call read id=c1 args={} said]`, nil, false},
		{"arguments not an object", `[tool call read id=c1 args=[1]]`, nil, false},
	}
	for _, c := range cases {
		calls, _, ok := parseTextCalls(c.in, names)
		if ok != c.ok {
			t.Fatalf("%s: ok %v, want %v", c.name, ok, c.ok)
		}
		if !ok {
			continue
		}
		if len(calls) != len(c.calls) {
			t.Fatalf("%s: %d calls, want %d", c.name, len(calls), len(c.calls))
		}
		for i, call := range calls {
			var v map[string]any
			if decodeJSON(call.Args, &v) != nil {
				t.Fatalf("%s: arguments unreadable: %s", c.name, call.Args)
			}
			b, _ := json.Marshal(v)
			if call.Name+" "+string(b) != c.calls[i] {
				t.Fatalf("%s: %s %s, want %s", c.name, call.Name, b, c.calls[i])
			}
		}
	}
}
