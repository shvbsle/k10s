package k8s

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func FormatAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

func FormatGVR(gvr schema.GroupVersionResource) string {
	resource := gvr.Resource
	if len(gvr.Group) > 0 {
		resource += "." + gvr.Group
	}
	resource += "/" + gvr.Version
	return resource
}
