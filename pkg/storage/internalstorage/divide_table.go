package internalstorage

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetTable return table name using gvr string
func GetTable(gvr schema.GroupVersionResource) string {
	group := strings.ReplaceAll(gvr.Group, ".", "_")
	return fmt.Sprintf("%s_%s_%s", group, gvr.Version, gvr.Resource)
}
