package resources

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/shvbsle/k10s/internal/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// allocContext holds pre-computed allocation bar strings keyed by node name.
// It is set before resolver execution and cleared after.
var allocContext map[string]string

// SetAllocContext sets the allocation context map used by the allocBar resolver.
func SetAllocContext(ctx map[string]string) {
	allocContext = ctx
}

// ClearAllocContext clears the allocation context map.
func ClearAllocContext() {
	allocContext = nil
}

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
	"instanceType": func(obj *unstructured.Unstructured) (string, error) {
		it := k8s.InstanceType(obj)
		if it == "" {
			return "—", nil
		}
		return it, nil
	},
	"computeType": func(obj *unstructured.Unstructured) (string, error) {
		class := k8s.ClassifyNode(obj)
		switch class {
		case k8s.NodeClassGPU_NVIDIA:
			return fmt.Sprintf("gpu/nvidia %d×", k8s.GPUCapacity(obj)), nil
		case k8s.NodeClassGPU_Neuron:
			return fmt.Sprintf("gpu/neuron %d×", k8s.GPUCapacity(obj)), nil
		default:
			return "cpu", nil
		}
	},
	"allocBar": func(obj *unstructured.Unstructured) (string, error) {
		if allocContext == nil {
			return "—", nil
		}
		if val, ok := allocContext[obj.GetName()]; ok {
			return val, nil
		}
		return "—", nil
	},
	"nodeStatus": func(obj *unstructured.Unstructured) (string, error) {
		return k8s.NodeReadyStatus(obj), nil
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
