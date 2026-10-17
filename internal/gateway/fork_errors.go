package gateway

// The fork's customer copy, mounted apart from the upstream files it
// speaks for: everything this fork says to a paying stranger in Chinese
// lives in this file, which upstream never has, so a nightly merge of
// upstream/main cannot conflict with it. What it says follows three
// rules (057da9c2):
//
//  1. a message magpie itself composes for a paying stranger says what
//     happened to the service and what to do next, in Chinese, and
//     nothing else: no account address, no provider or key name, no
//     share of a quota, no Settings page or CLI command. The details
//     stay in the log, put there where the message is composed.
//  2. a vendor's own error body passes as the vendor wrote it.
//  3. no message says the stack's name: toldOutside, below, is the net
//     under every reply, masking an address and the name wherever one
//     slipped through.
//
// The *Told twins here wrap the upstream functions that hold the
// English detail (cappedError, sealedTaskError, ...), log that detail
// for the operator, and answer in Chinese. Their call sites in the
// shared files are a line each, which a merge passes over; when
// upstream grows a new customer-facing message, the twin for it is
// added here and the call site switched, nothing else moved.

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/gwbudget"
	"github.com/yetone/magpie/internal/provider"
)

// cutTold is what a customer is told of a cut, in Chinese, no provider
// named: which provider lost the connection, and why, are the log's to
// hold. The failure's own words — cutMidReply's, the ended-short's —
// are what routing and the failure's class read on.
const (
	cutToldNoConn = "模型服务暂时无法连接，请稍后重试"
	cutToldMid    = "回复中途连接中断，请重试"
	cutToldShort  = "回复提前结束，请重试"
)

// toldOutside masks what an error body told to a paying stranger must not
// say: an account's address, and the stack's own name. A marker the
// gateway reads back out of a body (codexResetSaid, the lapse words
// ClaudeSignInRequired matches) must be written to survive this — no
// address, no "magpie".
var (
	outsideEmail = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	outsideStack = regexp.MustCompile(`(?i)\bmagpie('s)?\b`)
)

func toldOutside(msg string) string {
	msg = outsideEmail.ReplaceAllString(msg, "***@***")
	return outsideStack.ReplaceAllString(msg, "the gateway$1")
}

// untilInWords says a wait the way a customer reads it, not by the clock of
// the machine serving them.
func untilInWords(d time.Duration) string {
	if d < time.Minute {
		return "1 分钟内"
	}
	if m := int(d.Minutes()) + 1; m <= 90 {
		return fmt.Sprintf("约 %d 分钟", m)
	}
	return fmt.Sprintf("约 %.1f 小时", d.Hours())
}

// writeCut answers the agent whose request never reached the vendor: the
// gateway's own connection to it failed before a reply began. The agent is
// told 504, in Chinese, no provider named — a relay in front of this one
// (new-api) sends a 504 straight back to the agent rather than over to
// another provider (504 is in its never-retry set), so the agent's own
// retry comes here again; the details stay in the log. The failure's own
// class — 502, the connection error and its words — is what routing, rest
// and the failure's class read on, unchanged: where the reply is held for
// another candidate to answer, the 502 and the error's words are held as
// they were (code(), errBody()) and the 504 is said only when the held
// error is released (holdWriter.release); where nothing is held, and for a
// count, it is said at once.
func writeCut(w http.ResponseWriter, proto provider.Protocol, name string, err error) int {
	log.Printf("gateway: %s: no connection to the vendor: %v", name, err)
	if hw, ok := w.(*holdWriter); ok && hw.hold {
		hw.cutBody = cutBodyOf(proto)
		return writeError(w, proto, http.StatusBadGateway, name+": "+err.Error())
	}
	writeError(w, proto, http.StatusGatewayTimeout, cutToldNoConn)
	return http.StatusBadGateway
}

// cutBodyOf is the 504 error body writeCut's answer goes to the agent as,
// in the protocol's own shape, as writeError builds one.
func cutBodyOf(proto provider.Protocol) []byte {
	var v any
	switch proto {
	case provider.Anthropic:
		v = map[string]any{"type": "error", "error": map[string]any{"type": "api_error", "message": cutToldNoConn}}
	case provider.Gemini:
		v = map[string]any{"error": map[string]any{"code": http.StatusGatewayTimeout, "message": cutToldNoConn, "status": "UNAVAILABLE"}}
	default:
		v = map[string]any{"error": map[string]any{"message": cutToldNoConn, "type": "api_error", "code": nil, "param": nil}}
	}
	b, _ := json.Marshal(v)
	return b
}

// cappedErrorTold is cappedError as a paying stranger reads it: the
// service is out and when it is back, in Chinese, no account named and
// no share of a quota; what holds each account stays in the log.
func cappedErrorTold(model string, ws []Weighed, now time.Time) (string, time.Time) {
	held, soonest := heldReasons(ws, now)
	if len(held) > 0 {
		log.Printf("usage cap: %s: %s", model, strings.Join(held, "; "))
	}
	back := ""
	if !soonest.IsZero() {
		back = "，预计" + untilInWords(time.Until(soonest)) + "后恢复"
	}
	return fmt.Sprintf("模型 %s 的服务额度暂时用尽%s，请稍后重试", model, back), soonest
}

// checkpointCappedErrorTold is checkpointCappedError as cappedErrorTold
// is cappedError: the one account a natively-compacted conversation can
// go to is held, and another account would refuse the compaction it
// can't read — said in Chinese to the customer, the sealer in the log
// alone.
func checkpointCappedErrorTold(model string, cp checkpointWriter, ws []Weighed, now time.Time) (string, time.Time) {
	held, soonest := heldReasons(ws, now)
	if len(held) > 0 {
		log.Printf("usage cap: %s: sealed by %s: %s", model, cp.Who, strings.Join(held, "; "))
	}
	back := ""
	if !soonest.IsZero() {
		back = "，预计" + untilInWords(time.Until(soonest)) + "后恢复"
	}
	return fmt.Sprintf("模型 %s 的服务额度暂时用尽%s；该会话的上下文已绑定当前通道，切换通道会无法读取，请稍后在原会话重试，或开启新会话", model, back), soonest
}

// sealedTaskTold is sealedTaskError as a paying stranger reads it: the
// task is sealed and unreadable, said in Chinese; which server sealed
// it and the operator's way around it (the upstream message, with its
// Settings advice) stay in the log.
func sealedTaskTold(model, lead, earlier string) string {
	log.Printf("sealed task: model=%s lead=%s earlier=%s: %s", model, lead, earlier, sealedTaskError(model, lead, earlier))
	return fmt.Sprintf("该子任务的内容已由上游加密绑定（sealed），只有当初生成它的服务能解读，当前模型（%s）无法读取。请让发起方在与原任务一致的模型下重新下发该子任务。", model)
}

// keyLimitTold is what a paying stranger is told when the gateway key
// their request came on spent its window: the limit is met and when it
// lifts, in Chinese; the key's name and numbers are the operator's and
// stay in the log (keyLimited writes the refusal's own words there).
func keyLimitTold(refused gwbudget.Refusal, now time.Time) string {
	return fmt.Sprintf("该密钥本周期用量已达上限，预计%s后重置，请稍后重试", untilInWords(time.Duration(refused.RetryAfter(now))*time.Second))
}

// keyLimitOffTold is keyLimitTold for a key held to a limit no limiter
// serves: nothing was sent, and whose to fix is the operator's to read
// (upstream's ErrUnlimited says the Settings path, to the log alone).
func keyLimitOffTold() string {
	return "当前访问密钥暂不可用，请稍后重试或更换密钥"
}
