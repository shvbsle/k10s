package resources

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/shvbsle/k10s/internal/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var resolverMap = map[string]func(*unstructured.Unstructured) (string, error){
	"age": func(object *unstructured.Unstructured) (string, error) {
		return k8s.FormatAge(object.GetCreationTimestamp().Time), nil
	},
}

// Resolver defines methods for resolving field data from a kubernetes object
type Resolver struct {
	// PathTemplate is a go template that gets executed on an unstructured
	// object.
	PathTemplate string `json:"path"`

	// FuncName is a predefined function helper that can be applied to an
	// unstructured object.
	FuncName string `json:"func"`

	// CELExpression is a CEL statement that gets executed on the unstructured
	// kubernetes object.
	CELExpression string `json:"cel"`
}

func (r Resolver) Resolve(object *unstructured.Unstructured) (string, error) {
	// the first successful resolution is used, with the priority on sources
	// being dictated by the ordering below.

	if len(r.FuncName) > 0 {
		if fn, ok := resolverMap[r.FuncName]; ok {
			return fn(object)
		}
	}

	if len(r.PathTemplate) > 0 {
		var fieldBuffer bytes.Buffer
		if err := template.Must(template.New("").Parse(r.PathTemplate)).Execute(&fieldBuffer, object.UnstructuredContent()); err != nil {
			return "", err
		}
		return fieldBuffer.String(), nil
	}

	return "", fmt.Errorf("failed to resolve object field: %+v", object.UnstructuredContent())
}
