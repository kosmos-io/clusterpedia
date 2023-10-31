package internalstorage

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetShardingTable(gvr schema.GroupVersionResource) string {
	group := strings.ReplaceAll(gvr.Group, ".", "_")
	return fmt.Sprintf("%s_%s_%s", group, gvr.Version, gvr.Resource)
}
