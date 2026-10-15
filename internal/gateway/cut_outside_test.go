package gateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// The agent is told 504, in Chinese, no provider named; the failure's own
// class (502) is what routing and rest read on.
func TestWriteCutTells504KeepsTheClass(t *testing.T) {
	w := httptest.NewRecorder()
	got := writeCut(w, provider.Anthropic, "the-name", errors.New("dial tcp: connection refused"))
	if got != http.StatusBadGateway {
		t.Fatalf("the failure's class changed: %d, want 502", got)
	}
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("the agent was told %d, want 504: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "模型服务暂时无法连接") {
		t.Fatalf("the agent was told: %s", w.Body)
	}
	if strings.Contains(w.Body.String(), "the-name") || strings.Contains(w.Body.String(), "refused") {
		t.Fatalf("details leaked to the agent: %s", w.Body)
	}
}
