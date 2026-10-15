package gateway

import (
	"fmt"
	"strings"

	"github.com/yetone/magpie/internal/provider"
)

// unknownModel is what a request for a model the gateway doesn't know is
// told: the id asked for, the nearest ids to it, and to pick from the
// list — not every id there is, nor how a model may be named, which is
// the operator's business.
func unknownModel(asked, what string) string {
	msg := fmt.Sprintf("不支持的模型：%q%s", asked, what)
	if near := provider.Closest(asked, 3); len(near) > 0 {
		msg += "；相近的模型：" + strings.Join(near, ", ")
	}
	msg += "。请检查模型名，或从可用模型列表中选择"
	return msg
}
