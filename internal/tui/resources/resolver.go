package resources

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/shvbsle/k10s/internal/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// resolverMap maps function names (referenced in resource.views.json) to their
// implementations. Each function extracts a display value from an unstructured
// Kubernetes object.
var resolverMap = map[string]func(*unstructured.Unstructured) (string, error){
	k8s.ResolverFuncAge: func(obj *unstructured.Unstructured) (string, error) {
		return k8s.FormatAge(obj.GetCreationTimestamp().Time), nil
	},
	"podStatus": func(obj *unstructured.Unstructured) (string, error) {
		return k8s.PodDisplayStatus(obj), nil
	},
	"restarts": func(obj *unstructured.Unstructured) (string, error) {
		return k8s.PodRestartCount(obj), nil
	},
}

// Resolver defines methods for resolving field data from a Kubernetes object.
// Exactly one of PathTemplate, FuncName, or CELExpression should be set.
type Resolver struct {
	// PathTemplate is a Go template executed against the unstructured object.
	PathTemplate string `json:"path"`

	// FuncName references a predefined function in resolverMap.
	FuncName string `json:"func"`

	// CELExpression is a CEL statement executed against the unstructured object.
	CELExpression string `json:"cel"`
}

// Resolve returns the display string for the given object. It tries FuncName
// first, then PathTemplate, returning an error if neither is configured.
func (r Resolver) Resolve(object *unstructured.Unstructured) (string, error) {
	if r.FuncName != "" {
		if fn, ok := resolverMap[r.FuncName]; ok {
			return fn(object)
		}
	}

	if r.PathTemplate != "" {
		var buf bytes.Buffer
		tmpl, err := template.New("").Parse(r.PathTemplate)
		if err != nil {
			return "", fmt.Errorf("invalid path template %q: %w", r.PathTemplate, err)
		}
		if err := tmpl.Execute(&buf, object.UnstructuredContent()); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	return "", fmt.Errorf("resolver has no func or path configured")
}
