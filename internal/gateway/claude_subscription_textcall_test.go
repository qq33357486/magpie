package gateway

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A model deep into a conversation a run is told again may write its calls
// the way it reads them made (renderParts) instead of making them, and the
// client gets text it can do nothing with. The subscription's relay makes
// them the calls they stand for, and the turn ends as one that called
// tools, so the agent goes on instead of the session being lost.
func TestSubscriptionTextCalls(t *testing.T) {
	lines := []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_tc","model":"claude-sonnet-5-5","usage":{"input_tokens":2,"output_tokens":1}}}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me look.\n[tool ca"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ll mcp__magpie__read id=c1 args={\"path\":\"/tmp/x\"}]"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`,
		`{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":9}}}`,
		`{"type":"stream_event","event":{"type":"message_stop"}}`,
		`{"type":"result","subtype":"success","usage":{"input_tokens":2,"output_tokens":9}}`,
	}
	reply := func(t *testing.T, stream bool, tools ...string) string {
		t.Helper()
		run := &subscriptionRun{}
		segment := run.attach()
		pr, pw := io.Pipe()
		go run.readOutput(pr)
		go func() {
			for _, l := range lines {
				pw.Write([]byte(l + "\n"))
			}
			pw.Close()
		}()
		t.Cleanup(func() { pw.Close() })
		rec := httptest.NewRecorder()
		req := &Request{Model: "claude-sonnet-5-5", Stream: stream}
		for _, n := range tools {
			req.Tools = append(req.Tools, Tool{Name: n})
		}
		var usage Usage
		relay(rec, httptest.NewRequest("POST", "/v1/messages", nil), provider.Anthropic, "Claude Code", req, segment, &usage,
			func() {}, nil, func(string, string, bool) {})
		return rec.Body.String()
	}

	// streamed, the mark cut across two deltas: the call comes back as a
	// tool_use block, the text before it stands, and the turn ends as one
	// that called tools
	raw := reply(t, true, "read", "bash")
	if !strings.Contains(raw, `"type":"tool_use"`) || !strings.Contains(raw, `"name":"read"`) {
		t.Fatalf("no call made of the text:\n%s", raw)
	}
	if !strings.Contains(raw, `"stop_reason":"tool_use"`) {
		t.Fatalf("the turn didn't end as one that called tools:\n%s", raw)
	}
	if !strings.Contains(raw, "Let me look.") || strings.Contains(raw, "[tool call") {
		t.Fatalf("the text:\n%s", raw)
	}
	if !strings.Contains(raw, `"call_text_`) {
		t.Fatalf("the call's id:\n%s", raw)
	}

	// told whole: the same
	raw = reply(t, false, "read", "bash")
	if !strings.Contains(raw, `"type":"tool_use"`) || !strings.Contains(raw, `"name":"read"`) || !strings.Contains(raw, `"stop_reason":"tool_use"`) {
		t.Fatalf("no call made of the text:\n%s", raw)
	}

	// a tool the request didn't offer: the text goes as it came
	raw = reply(t, true, "bash")
	if strings.Contains(raw, `"type":"tool_use"`) || !strings.Contains(raw, "[tool call") {
		t.Fatalf("a call of a tool not offered:\n%s", raw)
	}
}
