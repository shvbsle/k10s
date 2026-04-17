package k8s

import (
	"fmt"
	"math"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResolverFuncAge is the canonical name for the age resolver function,
// used in resource view JSON configs and looked up by the refresh tick.
const ResolverFuncAge = "age"

// FormatAge returns a human-readable duration string relative to now.
// It uses the same compact format as kubectl: "30s", "5m", "3h", "7d".
func FormatAge(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(math.Round(d.Seconds())))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// FormatGVR returns a human-readable string for a GroupVersionResource,
// e.g. "pods/v1" or "deployments.apps/v1".
func FormatGVR(gvr schema.GroupVersionResource) string {
	resource := gvr.Resource
	if len(gvr.Group) > 0 {
		resource += "." + gvr.Group
	}
	resource += "/" + gvr.Version
	return resource
}
