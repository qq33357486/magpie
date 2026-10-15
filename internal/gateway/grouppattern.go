package gateway

import (
	"fmt"
	"strings"

	"github.com/yetone/magpie/internal/provider"
)

// emptyGroup is the user's group an id names when it has no member now:
// one whose patterns match no model magpie serves (#766).
func emptyGroup(id string) (provider.Group, bool) {
	gid, ok := strings.CutPrefix(id, provider.GroupPrefix)
	if !ok {
		return provider.Group{}, false
	}
	for _, g := range provider.Groups() {
		if strings.EqualFold(g.ID, gid) && len(g.Members) == 0 {
			return g, true
		}
	}
	return provider.Group{}, false
}

func emptyGroupError(g provider.Group) string {
	return fmt.Sprintf("分组 %s 当前没有可用模型，请更换其他模型", g.Name)
}
