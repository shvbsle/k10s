package resources

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/table"
	"github.com/shvbsle/k10s/internal/k8s"
)

const (
	resourceSchemaFileName = "resource.views.json"
)

//go:embed resource.views.json
var defaultResourceViews []byte

var resourceViews ResourceViews

func init() {
	resourceViewsJson := defaultResourceViews
	if home, err := os.UserHomeDir(); err == nil {
		if schema, err := os.ReadFile(filepath.Join(home, resourceSchemaFileName)); err == nil {
			resourceViewsJson = schema
		}
	}
	if err := json.Unmarshal(resourceViewsJson, &resourceViews); err != nil {
		panic(err)
	}
}

func GetColumns(totalWidth int, resource k8s.ResourceType) []table.Column {
	// Special handling for contexts view
	if resource == "contexts" {
		return []table.Column{
			{Title: "Context", Width: int(float32(totalWidth) * 0.8)},
			{Title: "Current", Width: int(float32(totalWidth) * 0.2)},
		}
	}

	var columns []table.Column
	for _, field := range GetResourceView(resource).Fields {
		columns = append(columns, table.Column{
			Title: field.Name,
			Width: int(field.Weight * float32(totalWidth)),
		})
	}
	return columns
}

// TODO: make this by GVR?
func GetResourceView(resource k8s.ResourceType) ResourceView {
	if view, ok := resourceViews[resource]; ok {
		return view
	}
	return ResourceView{
		Fields: []ResourceViewField{
			// TODO: this effectively assumed the resource is namespaced, but we
			// can always determine that dynamically using the API.
			{
				Name: "Namespace",
				Resolver: Resolver{
					PathTemplate: "{{ .metadata.namespace }}",
				},
				Weight: .3,
			},
			{
				Name: "Name",
				Resolver: Resolver{
					PathTemplate: "{{ .metadata.name }}",
				},
				Weight: .5,
			},
			{
				Name: "Age",
				Resolver: Resolver{
					FuncName: "age",
				},
				Weight: .2,
			},
		},
	}
}

type ResourceViews map[k8s.ResourceType]ResourceView

type ResourceView struct {
	DrillDown *struct {
		Resource          string   `json:"resource"`
		SelectorTemplates []string `json:"selectors"`
	} `json:"drill,omitempty"`
	Fields []ResourceViewField `json:"fields"`
}

type ResourceViewField struct {
	Weight   float32  `json:"weight"`
	Resolver Resolver `json:"resolver"`
	Name     string   `json:"name"`
}
