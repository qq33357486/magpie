package gateway

import (
	"bytes"
	"encoding/json"

	"github.com/tidwall/gjson"
)

// A drawing asked of the ChatGPT backend goes with stream: answered as one
// JSON, its connection sits silent for the whole minute or two a generation
// takes, and through the proxy an account draws over, one that says nothing
// that long is cut as dead — the EOF a relay's callers drew, every failure
// a couple of minutes in and none once the ask streamed (2026-10-14).
// Streamed, the partial images keep the connection spoken to while it draws.
//
// imagesAnswerOfStream is that streamed answer as the JSON readImagesAnswer
// reads: every completed event's image, and the usage the last of them
// named — the Images API's own events, or a Responses-shaped stream's
// image_generation_call items. b itself when it isn't a stream at all: a
// vendor that answered plain JSON anyway, or an error's page.
func imagesAnswerOfStream(b []byte) []byte {
	head := bytes.TrimSpace(b)
	if !bytes.HasPrefix(head, []byte("data:")) && !bytes.HasPrefix(head, []byte("event:")) && !bytes.HasPrefix(head, []byte(":")) {
		return b // a JSON answer begins '{', an error's page '<'
	}
	var answer struct {
		Data  []map[string]string `json:"data"`
		Usage struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
	}
	for _, line := range bytes.Split(b, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue // an event's name, a comment, or a blank line
		}
		d := bytes.TrimSpace(line[len("data:"):])
		if len(d) == 0 || d[0] == '[' {
			continue // [DONE], and whatever else a stream frames with
		}
		switch gjson.GetBytes(d, "type").String() {
		case "image_generation.completed":
			img := map[string]string{}
			put(img, "b64_json", firstOf(d, "b64_json", "result"))
			put(img, "url", strOf(d, "url"))
			put(img, "revised_prompt", strOf(d, "revised_prompt"))
			put(img, "media_type", strOf(d, "media_type"))
			answer.Data = append(answer.Data, img)
			i, o := usageOf(gjson.GetBytes(d, "usage"))
			answer.Usage.Input, answer.Usage.Output = max(answer.Usage.Input, i), max(answer.Usage.Output, o)
		case "response.completed":
			gjson.GetBytes(d, "response.output").ForEach(func(_, item gjson.Result) bool {
				if item.Get("type").String() == "image_generation_call" {
					img := map[string]string{}
					put(img, "b64_json", item.Get("result").String())
					put(img, "revised_prompt", item.Get("revised_prompt").String())
					answer.Data = append(answer.Data, img)
				}
				return true
			})
			i, o := usageOf(gjson.GetBytes(d, "response.usage"))
			answer.Usage.Input, answer.Usage.Output = max(answer.Usage.Input, i), max(answer.Usage.Output, o)
		}
	}
	answer.Data = dropEmpty(answer.Data)
	if len(answer.Data) == 0 {
		return b // readImagesAnswer says what it couldn't read of it
	}
	out, err := json.Marshal(answer)
	if err != nil {
		return b
	}
	return out
}

// usageOf is the tokens a stream's event named, in the Images API's words
// or the Responses' (completion as output).
func usageOf(v gjson.Result) (int, int) {
	i := int(v.Get("input_tokens").Int() + v.Get("prompt_tokens").Int())
	o := int(v.Get("output_tokens").Int() + v.Get("completion_tokens").Int())
	return i, o
}

func firstOf(b []byte, keys ...string) string {
	for _, k := range keys {
		if s := gjson.GetBytes(b, k); s.Type == gjson.String && s.Str != "" {
			return s.Str
		}
	}
	return ""
}

func strOf(b []byte, key string) string {
	if s := gjson.GetBytes(b, key); s.Type == gjson.String {
		return s.Str
	}
	return ""
}

func put(m map[string]string, key, v string) {
	if v != "" {
		m[key] = v
	}
}

// dropEmpty keeps the images that carry one: a b64 or a URL to take.
func dropEmpty(images []map[string]string) []map[string]string {
	var out []map[string]string
	for _, img := range images {
		if img["b64_json"] != "" || img["url"] != "" {
			out = append(out, img)
		}
	}
	return out
}
