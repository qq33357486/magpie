package gateway

import (
	"strings"
	"testing"
)

// A client that takes the images out of its history (#1382) leaves Claude
// Code's notes of them alone in it; a run started anew, told the
// conversation again, is to read them as images shown earlier, not as ones
// sent with the turn to answer whose files are missing.
func TestImageNotesShownEarlier(t *testing.T) {
	rendered := func(t *testing.T, msgs []Message) (string, int) {
		blocks, err := renderClaudePrompt(&Request{Messages: msgs})
		if err != nil {
			t.Fatal(err)
		}
		var all strings.Builder
		images := 0
		for _, b := range blocks {
			if b["type"] == "image" {
				images++
				all.WriteString("<IMG>")
				continue
			}
			s, _ := b["text"].(string)
			all.WriteString(s)
		}
		return all.String(), images
	}

	// the images taken out of the history: only the notes left in it
	text, images := rendered(t, []Message{
		{Role: "user", Parts: []Part{{Kind: Text, Text: "look at this screenshot"}}},
		{Role: "assistant", Parts: []Part{{Kind: Text, Text: "a cat\n[Image: source: /root/.claude/projects/p/images/1.png]"}}},
		{Role: "user", Parts: []Part{{Kind: Text, Text: "what color was it?"}}},
	})
	if !strings.Contains(text, "[Image: shown earlier]") || strings.Contains(text, "/root/.claude") || strings.Contains(text, "images/1.png") {
		t.Fatalf("a note left in the history:\n%s", text)
	}
	if images != 0 {
		t.Fatalf("an image block without data: %d", images)
	}

	// the images still carried: their blocks go as before, the notes told
	// the same whatever the request carries
	text, images = rendered(t, []Message{
		{Role: "user", Parts: []Part{{Kind: Text, Text: "look at this"}, Part{Kind: Image, MediaType: "image/png", Data: "iVBORw0KGgo="}}},
		{Role: "assistant", Parts: []Part{{Kind: Text, Text: "a cat\n[Image: source: /root/.claude/projects/p/images/1.png]"}}},
		{Role: "user", Parts: []Part{{Kind: Text, Text: "what color was it?"}}},
	})
	if !strings.Contains(text, "[Image: shown earlier]") || strings.Contains(text, "/root/.claude") {
		t.Fatalf("a note beside its image:\n%s", text)
	}
	if images != 1 {
		t.Fatalf("the image itself went missing: %d", images)
	}

	// only a line of a message's own is a note; one said in passing stays
	text, _ = rendered(t, []Message{
		{Role: "user", Parts: []Part{{Kind: Text, Text: "the note [Image: source: /x/1.png] was said in passing"}}},
		{Role: "assistant", Parts: []Part{{Kind: Text, Text: "ok"}}},
		{Role: "user", Parts: []Part{{Kind: Text, Text: "go on"}}},
	})
	if !strings.Contains(text, "[Image: source: /x/1.png]") {
		t.Fatalf("a note in passing:\n%s", text)
	}
}
